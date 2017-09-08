package jenkins

import (
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bytes"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/pipeline/notification"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/restfulserver"
	"github.com/sluu99/uuid"
)

type BuildStruct struct {
	Repository string
	Branch     string
	Workspace  string
	Command    string
}

type JenkinsProvider struct {
	pipeline *pipeline.Pipeline
}

func (j *JenkinsProvider) Init(pipeline *pipeline.Pipeline) error {
	j.pipeline = pipeline
	println("get in provider")
	return nil
}

func (j *JenkinsProvider) RunPipeline(p *pipeline.Pipeline) (*pipeline.Activity, error) {
	j.Init(p)

	//init and create  activity
	id := uuid.Rand().Hex()
	//Find a jenkins slave on which to run
	nodeName, err := getNodeNameToRun(id)
	if err != nil {
		return &pipeline.Activity{}, err
	}
	logrus.Infof("runpipeline,get nodeName:%v", nodeName)
	activity := pipeline.Activity{
		Id:          id,
		Pipeline:    *p,
		RunSequence: p.RunCount + 1,
		Status:      pipeline.ActivityWaiting,
		StartTS:     time.Now().UnixNano() / int64(time.Millisecond),
		NodeName:    nodeName,
	}
	for _, stage := range p.Stages {
		activity.ActivityStages = append(activity.ActivityStages, ToActivityStage(stage))
	}
	//logrus.Infof("creating activity:%v", activity)
	_, err = restfulserver.CreateActivity(activity)
	if err != nil {
		return &pipeline.Activity{}, err
	}
	if len(p.Stages) == 0 {
		return &pipeline.Activity{}, errors.New("no stage in pipeline definition to run!")
	}
	for i := 0; i < len(p.Stages); i++ {
		//logrus.Infof("creating stage:%v", p.Stages[i])
		if err := j.CreateStage(&activity, i); err != nil {
			logrus.Error(errors.Wrapf(err, "stage <%s> fail", p.Stages[i].Name))
			return &pipeline.Activity{}, err
		}
	}
	//logrus.Infof("running stage:%v", p.Stages[0])
	err = j.RunStage(&activity, 0)

	if err != nil {
		return &pipeline.Activity{}, err
	}

	return &activity, nil
}

//RerunActivity runs an existing activity
func (j *JenkinsProvider) RerunActivity(a *pipeline.Activity) error {
	//find an available node to run
	nodeName, err := getNodeNameToRun(a.Id)
	if err != nil {
		return err
	}
	a.NodeName = nodeName
	//set to original git commit
	err = j.SetSCMCommit(a)
	if err != nil {
		logrus.Errorf("set scm commit fail,%v", err)
	}

	logrus.Infof("rerunpipeline,get nodeName:%v", nodeName)
	a.RunSequence = a.Pipeline.RunCount + 1
	a.StartTS = time.Now().UnixNano() / int64(time.Millisecond)
	err = j.RunStage(a, 0)
	return err
}

//CreateStage init jenkins project settings of the stage
func (j *JenkinsProvider) CreateStage(activity *pipeline.Activity, ordinal int) error {
	logrus.Info("create jenkins job from stage")
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	jobName := j.pipeline.Name + "_" + stage.Name + "_" + activityId

	conf := j.generateJenkinsProject(activity, ordinal)

	bconf, _ := xml.MarshalIndent(conf, "  ", "    ")
	if err := CreateJob(jobName, bconf); err != nil {
		return err
	}
	return nil
}

//getNodeNameToRun gets a specific slave name to run given activity id
func getNodeNameToRun(id string) (string, error) {
	nodes, err := GetActiveNodesName()
	if err != nil || len(nodes) == 0 {
		return "", errors.Wrapf(err, "fail to find an active slave to work")
	}
	//hash to one of the nodes
	h := fnv.New32a()
	h.Write([]byte(id))
	index := int(h.Sum32()) % len(nodes)
	return nodes[index], nil
}

//DeleteFormerBuild delete last build info of a completed activity
func (j *JenkinsProvider) DeleteFormerBuild(activity *pipeline.Activity) error {
	if activity.Status == pipeline.ActivityBuilding || activity.Status == pipeline.ActivityWaiting {
		return errors.New("cannot delete lastbuild of running activity!")
	}
	activityId := activity.Id
	for _, stage := range activity.ActivityStages {
		jobName := activity.Pipeline.Name + "_" + stage.Name + "_" + activityId
		if stage.Status == pipeline.ActivityStageSuccess || stage.Status == pipeline.ActivityStageFail {
			logrus.Infof("deleting:%v", jobName)
			err := DeleteBuild(jobName)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

//SetSCMCommit update jenkins job SCM to use commit id in activity
func (j *JenkinsProvider) SetSCMCommit(activity *pipeline.Activity) error {
	if activity.CommitInfo == "" {
		return errors.New("no commit info in activity")
	}
	conf := j.generateJenkinsProject(activity, 0)
	conf.Scm.GitBranch = activity.CommitInfo

	stageName := activity.ActivityStages[0].Name
	jobName := activity.Pipeline.Name + "_" + stageName + "_" + activity.Id

	bconf, _ := xml.MarshalIndent(conf, "  ", "    ")
	logrus.Infof("conf:\n%v", string(bconf))
	logrus.Infof("trying to set commit")
	if err := UpdateJob(jobName, bconf); err != nil {
		logrus.Infof("updatejob error:%v", err)
		return err
	}
	return nil
}

func (j *JenkinsProvider) RunStage(activity *pipeline.Activity, ordinal int) error {
	logrus.Info("begin jenkins stage")
	//logrus.Infof("hi,%v\nhi,%v\nhi,%v\nhi,%v", activity.Pipeline, activity, len(activity.Pipeline.Stages), ordinal)
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	jobName := activity.Pipeline.Name + "_" + stage.Name + "_" + activityId

	if _, err := BuildJob(jobName, map[string]string{}); err != nil {
		return err
	}

	return nil
}

func (j *JenkinsProvider) RunBuild(stage *pipeline.Stage, activityId string) error {
	return nil
}

func (j *JenkinsProvider) generateJenkinsProject(activity *pipeline.Activity, ordinal int) *JenkinsProject {
	logrus.Info("generating jenkins project config")
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	workspaceName := path.Join("${JENKINS_HOME}", "workspace", activityId)

	taskShells := []JenkinsTaskShell{}
	for stepOrdinal, step := range stage.Steps {
		step.Services = pipeline.GetServices(activity, ordinal, stepOrdinal)
		taskShells = append(taskShells, JenkinsTaskShell{Command: commandBuilder(activity, step)})
	}
	commandBuilders := JenkinsBuilder{TaskShells: taskShells}

	scm := JenkinsSCM{Class: "hudson.scm.NullSCM"}
	if len(stage.Steps) == 1 && stage.Steps[0].Type == pipeline.StepTypeSCM {
		scm = JenkinsSCM{
			Class:         "hudson.plugins.git.GitSCM",
			Plugin:        "git@3.3.1",
			ConfigVersion: 2,
			GitRepo:       stage.Steps[0].Repository,
			GitBranch:     stage.Steps[0].Branch,
		}
	}
	v := &JenkinsProject{
		Scm:          scm,
		AssignedNode: activity.NodeName,
		CanRoam:      false,
		Disabled:     false,
		BlockBuildWhenDownstreamBuilding: false,
		BlockBuildWhenUpstreamBuilding:   false,
		CustomWorkspace:                  workspaceName,
		Builders:                         commandBuilders,
		BuildWrappers:                    TimestampWrapperPlugin{Plugin: "timestamper@1.8.8"},
	}
	//logrus.Infof("needapprove:%v,ordinal:%v", stage.NeedApprove, ordinal)
	if !stage.NeedApprove && ordinal > 0 {
		//Add build trigger
		prevJobName := j.pipeline.Name + "_" + j.pipeline.Stages[ordinal-1].Name + "_" + activityId
		v.Triggers = JenkinsTrigger{
			BuildTrigger: JenkinsBuildTrigger{
				UpstreamProjects:       prevJobName,
				ThresholdName:          "SUCCESS",
				ThresholdOrdinal:       0,
				ThresholdColor:         "BLUE",
				ThresholdCompleteBuild: true,
			},
		}
	}

	return v

}

func commandBuilder(activity *pipeline.Activity, step *pipeline.Step) string {
	stringBuilder := new(bytes.Buffer)
	switch step.Type {
	case pipeline.StepTypeTask:

		envVars := ""
		if len(step.Parameters) > 0 {
			for _, para := range step.Parameters {
				envVars += fmt.Sprintf("-e %s ", para)
			}
		}

		entrypointPara := ""
		argsPara := ""
		svcPara := ""
		if step.IsShell {
			entrypointPara = "--entrypoint /bin/sh"
			argsPara = ".r_cicd_entrypoint.sh"

			//write to a sh file,then docker run it
			stringBuilder.WriteString("cat>.r_cicd_entrypoint.sh<<EOF\n")
			cmd := strings.Replace(step.Command, "\\", "\\\\", -1)
			cmd = strings.Replace(cmd, "$", "\\$", -1)
			stringBuilder.WriteString(cmd)
			stringBuilder.WriteString("\nEOF\n")
			stringBuilder.WriteString("set -xe\n")
		} else {
			stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
			argsPara = step.Args
		}
		if step.Entrypoint != "" {
			entrypointPara = "--entrypoint " + step.Entrypoint
		}
		//isService
		if step.IsService {
			containerName := activity.Id + step.Alias
			svcPara = "-d --name " + containerName
			//stringBuilder.WriteString(fmt.Sprintf("docker run -d --env-file ${PWD}/.r_cicd.env %s --name %s %s %s %s", envVars, containerName, entrypointPara, step.Image, command))
			//break
		}

		//add link service
		linkInfo := ""
		if len(step.Services) > 0 {
			linkInfo += ""
			for _, svc := range step.Services {
				linkInfo += fmt.Sprintf("--link %s:%s ", svc.ContainerName, svc.Name)
			}

		}

		volumeInfo := "--volumes-from ${HOSTNAME} -w ${PWD}"
		stringBuilder.WriteString("docker run --rm")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString("--env-file ${PWD}/.r_cicd.env")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(envVars)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(svcPara)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(volumeInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(entrypointPara)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(linkInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(step.Image)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(argsPara)
	case pipeline.StepTypeBuild:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		if step.SourceType == "sc" {
			buildPath := "."
			if step.DockerfilePath != "" {
				buildPath = step.DockerfilePath
			}
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" ")
			stringBuilder.WriteString(buildPath)
			stringBuilder.WriteString(";")
		} else if step.SourceType == "file" {
			stringBuilder.WriteString("echo \"")
			stringBuilder.WriteString(strings.Replace(step.Dockerfile, "\"", "\\\"", -1))
			stringBuilder.WriteString("\">.Dockerfile;")
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" -f .Dockerfile .;")
		}
		if step.PushFlag {
			sep := strings.Index(step.TargetImage, "/")
			registry := ""
			if sep != -1 {
				registry = step.TargetImage[:sep]
			}
			stringBuilder.WriteString("docker login --username ")
			stringBuilder.WriteString(step.UserName)
			stringBuilder.WriteString(" --password ")
			stringBuilder.WriteString(step.Password)
			stringBuilder.WriteString(" ")
			stringBuilder.WriteString(registry)
			stringBuilder.WriteString(";docker push ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(";")
		}
	case pipeline.StepTypeSCM:
		//run a context container to share environment variables
		/*volumeInfo := "--volumes-from ${HOSTNAME} -w ${PWD}"
		stringBuilder.WriteString("docker run --privileged --name ")
		stringBuilder.WriteString(activity.Id + "_context ")
		stringBuilder.WriteString("-e GIT_COMMIT=${GIT_COMMIT} ")
		stringBuilder.WriteString("-v /var/lib/docker:/var/lib/docker -d docker:dind")

		*/

		//write to a env file that provides the environment variables to use throughout the activity.
		stringBuilder.WriteString("GIT_BRANCH=$(echo $GIT_BRANCH|cut -d / -f 2)\n")
		stringBuilder.WriteString("cat>.r_cicd.env<<EOF\n")
		stringBuilder.WriteString("CICD_GIT_COMMIT=$GIT_COMMIT\n")
		stringBuilder.WriteString("CICD_GIT_PREVIOUS_COMMIT=$GIT_PREVIOUS_COMMIT\n")
		stringBuilder.WriteString("CICD_GIT_PREVIOUS_SUCCESSFUL_COMMIT=$GIT_PREVIOUS_SUCCESSFUL_COMMIT\n")
		stringBuilder.WriteString("CICD_GIT_BRANCH=$GIT_BRANCH\n")
		stringBuilder.WriteString("CICD_GIT_LOCAL_BRANCH=$GIT_LOCAL_BRANCH\n")
		stringBuilder.WriteString("CICD_GIT_URL=$GIT_URL\n")
		stringBuilder.WriteString("CICD_GIT_COMMITTER_NAME=$GIT_COMMITTER_NAME\n")
		stringBuilder.WriteString("CICD_GIT_AUTHOR_NAME=$GIT_AUTHOR_NAME\n")
		stringBuilder.WriteString("CICD_GIT_COMMITTER_EMAIL=$GIT_COMMITTER_EMAIL\n")
		stringBuilder.WriteString("CICD_GIT_AUTHOR_EMAIL=$GIT_AUTHOR_EMAIL\n")
		stringBuilder.WriteString("CICD_SVN_REVISION=$SVN_REVISION\n")
		stringBuilder.WriteString("CICD_SVN_URL=$SVN_URL\n")
		stringBuilder.WriteString("CICD_PIPELINE_NAME=" + activity.Pipeline.Name + "\n")
		stringBuilder.WriteString("CICD_PIPELINE_ID=" + activity.Pipeline.Id + "\n")
		stringBuilder.WriteString("CICD_TRIGGER_TYPE=\n")
		stringBuilder.WriteString("CICD_NODE_NAME=" + activity.NodeName + "\n")
		stringBuilder.WriteString("CICD_ACTIVITY_ID=" + activity.Id + "\n")
		stringBuilder.WriteString("CICD_ACTIVITY_SEQUENCE=" + strconv.Itoa(activity.RunSequence) + "\n")
		stringBuilder.WriteString("\nEOF\n")

	case pipeline.StepTypeUpgradeService:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		stringBuilder.WriteString("docker run reg.cnrancher.com/rancher/rancher-upgrader:dev service")
		if step.Tag != "" {
			stringBuilder.WriteString(" --image ")
			stringBuilder.WriteString(step.Tag)
		}
		if step.BatchSize > 0 {
			stringBuilder.WriteString(" --batchsize ")
			stringBuilder.WriteString(strconv.Itoa(step.BatchSize))
		}
		if step.Interval != 0 {
			stringBuilder.WriteString(" --interval ")
			stringBuilder.WriteString(strconv.Itoa(step.Interval))
		}
		if step.StartFirst != false {
			stringBuilder.WriteString(" --startfirst")
			stringBuilder.WriteString(" true")
		}
		if step.DeployEnv == "others" {
			stringBuilder.WriteString(" --envurl ")
			stringBuilder.WriteString(step.Endpoint)
			stringBuilder.WriteString(" --accesskey ")
			stringBuilder.WriteString(step.Accesskey)
			stringBuilder.WriteString(" --secretkey ")
			stringBuilder.WriteString(step.Secretkey)
		} else if step.DeployEnv == "local" {
			//read from env var
			stringBuilder.WriteString(" --envurl $CATTLE_URL")
			stringBuilder.WriteString(" --accesskey $CATTLE_ACCESS_KEY")
			stringBuilder.WriteString(" --secretkey $CATTLE_SECRET_KEY")
		}
		for k, v := range step.ServiceSelector {
			stringBuilder.WriteString(" --selector ")
			stringBuilder.WriteString(k)
			stringBuilder.WriteString("=")
			stringBuilder.WriteString(v)
		}
	case pipeline.StepTypeUpgradeStack:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		if step.DeployEnv == "local" {
			script := fmt.Sprintf(upgradeStackScript, "$CATTLE_URL", "$CATTLE_ACCESS_KEY", "$CATTLE_SECRET_KEY", step.StackName, step.DockerCompose, step.RancherCompose)
			stringBuilder.WriteString(script)
		} else {
			script := fmt.Sprintf(upgradeStackScript, step.Endpoint, step.Accesskey, step.Secretkey, step.StackName, step.DockerCompose, step.RancherCompose)
			stringBuilder.WriteString(script)
		}
	case pipeline.StepTypeUpgradeCatalog:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")

		_, templateName, templateBase, _, _ := templateURLPath(step.ExternalId)

		systemFlag := ""
		if templateBase != "" {
			systemFlag = "--system "
		}
		deployFlag := ""
		if step.DeployFlag {
			deployFlag = "true"
		}

		dockerCompose := ""
		rancherCompose := ""
		readme := ""
		for _, pf := range step.FilesArray {
			if strings.HasPrefix(pf.Name, "docker-compose") {
				dockerCompose = pf.Body
			} else if strings.HasPrefix(pf.Name, "rancher-compose") {
				rancherCompose = pf.Body
			} else if pf.Name == "README.md" {
				readme = pf.Body
			}

		}
		dockerCompose = EscapeShell(dockerCompose)
		rancherCompose = EscapeShell(rancherCompose)
		readme = EscapeShell(readme)
		answers := step.Answers

		endpoint := step.Endpoint
		accessKey := step.Accesskey
		secretKey := step.Secretkey

		if step.DeployFlag && step.DeployEnv == "local" {
			endpoint = "$CATTLE_URL"
			accessKey = "$CATTLE_ACCESS_KEY"
			secretKey = "$CATTLE_SECRET_KEY"
		}
		script := fmt.Sprintf(upgradeCatalogScript, step.Repository, step.Branch, step.UserName, step.Password, systemFlag, templateName, deployFlag, dockerCompose, rancherCompose, readme, answers, endpoint, accessKey, secretKey, step.StackName)
		stringBuilder.WriteString(script)
	case pipeline.StepTypeDeploy:
	}
	//logrus.Infof("Finish building command for step command is %s", stringBuilder.String())
	return stringBuilder.String()
}

//SyncActivity gets latest activity info, return true if status if changed
func (j *JenkinsProvider) SyncActivity(activity *pipeline.Activity) (bool, error) {
	p := activity.Pipeline
	var updated bool

	//logrus.Infof("syncing activity:%v", activity.Id)
	//logrus.Infof("activity is:%v", activity)
	for i, actiStage := range activity.ActivityStages {
		jobName := p.Name + "_" + actiStage.Name + "_" + activity.Id
		beforeStatus := actiStage.Status

		if beforeStatus == pipeline.ActivityStageSuccess {
			continue
		}

		jobInfo, err := GetJobInfo(jobName)
		if err != nil {
			//cannot get jobinfo
			logrus.Infof("got job info:%v,err:%v", jobInfo, err)
			return false, err
		}

		/*
			if (jobInfo.LastBuild == JenkinsJobInfo.LastBuild{}) {
				//no build finish
				return nil
			}
		*/
		buildInfo, err := GetBuildInfo(jobName)
		//logrus.Infof("got build info:%v, err:%v", buildInfo, err)
		if err != nil {
			//cannot get build info
			//build not started
			if actiStage.Status == pipeline.ActivityStagePending {
				return updated, nil
			}
			actiStage.Status = pipeline.ActivityStageWaiting
			break
		}
		getCommit(activity, buildInfo)
		//if any buildInfo found,activity in building status
		activity.Status = pipeline.ActivityBuilding
		actiStage.Status = pipeline.ActivityStageBuilding
		actiStage.StartTS = buildInfo.Timestamp

		//logrus.Info("get buildinfo result:%v,actiStagestatus:%v", buildInfo.Result, actiStage.Status)
		if err == nil {
			rawOutput, err := GetBuildRawOutput(jobName)
			if err != nil {
				logrus.Infof("got rawOutput:%v,err:%v", rawOutput, err)
			}
			//actiStage.RawOutput = rawOutput
			stepStatusUpdated := parseSteps(actiStage, rawOutput)

			if actiStage.Status == pipeline.ActivityStageFail {
				activity.Status = pipeline.ActivityFail
				updated = true
			} else if actiStage.Status == pipeline.ActivityStageSuccess {
				if i == len(p.Stages)-1 {
					//if all stage success , mark activity as success
					activity.StopTS = buildInfo.Timestamp + buildInfo.Duration
					activity.Status = pipeline.ActivitySuccess
					updated = true
				}
				logrus.Infof("stage success:%v", i)

				if i < len(p.Stages)-1 && activity.Pipeline.Stages[i+1].NeedApprove {
					logrus.Infof("set pending")
					activity.Status = pipeline.ActivityPending
					activity.ActivityStages[i+1].Status = pipeline.ActivityStagePending
					activity.PendingStage = i + 1
				}
			}
			updated = updated || stepStatusUpdated
		}
		if beforeStatus != actiStage.Status {
			updated = true
			logrus.Infof("sync activity %v,updated !", activity.Id)
		}
		//logrus.Infof("after sync,beforestatus and after:%v,%v", beforeStatus, actiStage.Status)
	}

	return updated, nil
}

//OnActivityCompelte helps clean up
func (j *JenkinsProvider) OnActivityCompelte(activity *pipeline.Activity) {
	//clean services in activity
	services := pipeline.GetAllServices(activity)
	containerNames := []string{}
	for _, service := range services {
		containerNames = append(containerNames, service.ContainerName)
	}
	command := "docker rm -f " + strings.Join(containerNames, " ")
	cleanServiceScript := fmt.Sprintf(ScriptSkel, activity.NodeName, strings.Replace(command, "\"", "\\\"", -1))
	logrus.Infof("cleanservicescript is: %v", cleanServiceScript)
	res, err := ExecScript(cleanServiceScript)
	logrus.Infof("clean services result:%v,%v", res, err)

	//send notification
	if activity.Pipeline.IsEmail {
		if err := sendEmail(activity); err != nil {
			logrus.Errorf("Send Notification Email Error:%v", err)
		}

	}
	//clean workspace
	// command = "rm -rf ${System.getenv('JENKINS_HOME')}/workspace/" + activity.Id
	// cleanWorkspaceScript := fmt.Sprintf(ScriptSkel, activity.NodeName, strings.Replace(command, "\"", "\\\"", -1))
	// res, err = ExecScript(cleanWorkspaceScript)
	// logrus.Infof("clean workspace result:%v,%v", res, err)

}

func sendEmail(activity *pipeline.Activity) error {
	setting, err := restfulserver.GetPipelineSetting()
	if err != nil {
		return err
	}

	notifier := notification.NewEmailNotifier(setting.SMTPHost, setting.SMTPPort, setting.SMTPUsername, setting.SMTPPassword)
	stringbuilder := new(bytes.Buffer)
	stringbuilder.WriteString(activity.PipelineName)
	stringbuilder.WriteString(" - Execution #")
	stringbuilder.WriteString(strconv.Itoa(activity.RunSequence))
	stringbuilder.WriteString(" - ")
	stringbuilder.WriteString(activity.Status)
	stringbuilder.WriteString("!")

	subject := stringbuilder.String()
	tm := time.Unix(activity.StartTS, 0)
	tm.Format("2006-01-02 03:04:05 PM")
	stringbuilder.WriteString("\nStart time: ")
	stringbuilder.WriteString(tm.Format("2006-01-02 03:04:05 PM"))
	stringbuilder.WriteString("\nExecution time: ")
	duration := (activity.StopTS - activity.StartTS) / 1000
	stringbuilder.WriteString(strconv.Itoa(int(duration)))
	stringbuilder.WriteString("s\n")
	stringbuilder.WriteString("Status:\n")
	for _, stage := range activity.ActivityStages {
		stringbuilder.WriteString(stage.Name)
		stringbuilder.WriteString(": ")
		stringbuilder.WriteString(stage.Status)
		stringbuilder.WriteString("\n")
	}
	body := stringbuilder.String()

	return notifier.SendMail(activity.Pipeline.EmailRecipients, subject, body)
}

func (j *JenkinsProvider) GetStepLog(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) (string, error) {
	p := activity.Pipeline
	if stageOrdinal < 0 || stageOrdinal > len(activity.ActivityStages) || stepOrdinal < 0 || stepOrdinal > len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return "", errors.New("ordinal out of range")
	}
	actiStage := activity.ActivityStages[stageOrdinal]
	jobName := p.Name + "_" + actiStage.Name + "_" + activity.Id
	rawOutput, err := GetBuildRawOutput(jobName)
	if err != nil {
		return "", err
	}
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	outputs := regexp.MustCompile(token).Split(rawOutput, -1)
	if len(outputs) > 0 && stageOrdinal == 0 && stepOrdinal == 0 {
		// SCM
		return outputs[0], nil
	}
	if len(outputs) < stepOrdinal+2 {
		//no printed log
		return "", nil
	}
	//logrus.Infof("got step log:%v", outputs[stepOrdinal+1])
	return outputs[stepOrdinal+1], nil

}

func getCommit(activity *pipeline.Activity, buildInfo *JenkinsBuildInfo) {
	if activity.CommitInfo != "" {
		return
	}

	//logrus.Infof("try to get commitInfo,action:%v", buildInfo.Actions)
	actions := buildInfo.Actions
	for _, action := range actions {

		//logrus.Infof("lastbuiltrevision:%v", action.LastBuiltRevision.SHA1)
		if action.LastBuiltRevision.SHA1 != "" {
			activity.CommitInfo = action.LastBuiltRevision.SHA1
		}
	}
}

//parse jenkins rawoutput to steps,return true if status updated
func parseSteps(actiStage *pipeline.ActivityStage, rawOutput string) bool {
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	lastStatus := pipeline.ActivityStepBuilding
	var updated bool = false
	//TODO add timestamp
	if strings.HasSuffix(rawOutput, "  Finished: SUCCESS\n") {
		lastStatus = pipeline.ActivityStepSuccess
		actiStage.Status = pipeline.ActivityStageSuccess
	} else if strings.HasSuffix(rawOutput, "  Finished: FAILURE\n") {
		lastStatus = pipeline.ActivityStepFail
		actiStage.Status = pipeline.ActivityStageFail
	}
	//logrus.Infof("raw:%v,\ntoken:%v", rawOutput, token)
	outputs := regexp.MustCompile(token).Split(rawOutput, -1)
	//logrus.Infof("split to %v parts,steps number:%v, parse outputs:%v", len(outputs), len(actiStage.ActivitySteps), outputs)
	if len(outputs) > 0 && len(actiStage.ActivitySteps) > 0 && strings.Contains(outputs[0], "  Cloning the remote Git repository\n") {
		// SCM
		//actiStage.ActivitySteps[0].Message = outputs[0]
		if actiStage.ActivitySteps[0].Status != lastStatus {
			updated = true
			actiStage.ActivitySteps[0].Status = lastStatus
		}
		//get step time for SCM
		parseStepTime(actiStage.ActivitySteps[0], outputs[0], actiStage.StartTS)
		actiStage.Duration = actiStage.ActivitySteps[0].Duration
		return updated
	}
	logrus.Infof("parsed,len output:%v", len(outputs))
	stageTime := int64(0)
	for i, step := range actiStage.ActivitySteps {
		finishStepNum := len(outputs) - 1
		prevStatus := step.Status
		logrus.Info("getting step %v", i)
		if i < finishStepNum-1 {
			//passed steps
			//step.Message = outputs[i+1]
			step.Status = pipeline.ActivityStepSuccess
			parseStepTime(step, outputs[i+1], actiStage.StartTS)
			stageTime = stageTime + step.Duration
		} else if i == finishStepNum-1 {
			//last run step
			//step.Message = outputs[i+1]
			step.Status = lastStatus
			parseStepTime(step, outputs[i+1], actiStage.StartTS)
			stageTime = stageTime + step.Duration
		} else {
			//not run steps
			step.Status = pipeline.ActivityStepWaiting
		}
		if prevStatus != step.Status {
			updated = true
		}
		actiStage.ActivitySteps[i] = step
		logrus.Infof("now step is %v.", step)
	}
	actiStage.Duration = stageTime
	logrus.Infof("now actistage is %v.", actiStage)

	return updated

}

func parseStepTime(step *pipeline.ActivityStep, log string, activityStartTS int64) {
	//logrus.Infof("parsesteptime")
	token := "(^|\\n)\\w{14}  "
	r, _ := regexp.Compile(token)
	lines := r.FindAllString(log, -1)
	if len(lines) == 0 {
		return
	}

	start := strings.TrimLeft(lines[0], "\n")
	start = strings.TrimRight(start, " ")
	durationStart, err := time.ParseDuration(start)
	if err != nil {
		logrus.Errorf("parse duration error!%v", err)
		return
	}

	//compute step duration when done
	step.StartTS = activityStartTS + (durationStart.Nanoseconds() / int64(time.Millisecond))

	if step.Status != pipeline.ActivityStepSuccess && step.Status != pipeline.ActivityStepFail {
		return
	}

	end := strings.TrimLeft(lines[len(lines)-1], "\n")
	end = strings.TrimRight(end, " ")
	durationEnd, err := time.ParseDuration(end)
	if err != nil {
		logrus.Errorf("parse duration error!%vparseStepTime", err)
		return
	}
	duration := (durationEnd.Nanoseconds() - durationStart.Nanoseconds()) / int64(time.Millisecond)
	step.Duration = duration
}

func ToActivityStage(stage *pipeline.Stage) *pipeline.ActivityStage {
	actiStage := pipeline.ActivityStage{
		Name:          stage.Name,
		NeedApproval:  stage.NeedApprove,
		Status:        "Waiting",
		ActivitySteps: []*pipeline.ActivityStep{},
	}
	for _, step := range stage.Steps {
		actiStep := &pipeline.ActivityStep{
			Name:   step.Name,
			Status: pipeline.ActivityStepWaiting,
		}
		actiStage.ActivitySteps = append(actiStage.ActivitySteps, actiStep)
	}
	return &actiStage

}

func EscapeShell(script string) string {
	escaped := strings.Replace(script, "\\", "\\\\", -1)
	escaped = strings.Replace(escaped, "$", "\\$", -1)

	//TODO preserve a pattern
	for _, str := range pipeline.PreservedEnvs {
		escaped = strings.Replace(escaped, "\\$"+str+" ", "$"+str+" ", -1)
		escaped = strings.Replace(escaped, "\\$"+str+"\n", "$"+str+"\n", -1)
		escaped = strings.Replace(escaped, "\\${"+str+"}", "${"+str+"}", -1)

	}
	return escaped
}

func templateURLPath(path string) (string, string, string, string, bool) {
	pathSplit := strings.Split(path, ":")
	switch len(pathSplit) {
	case 2:
		catalog := pathSplit[0]
		template := pathSplit[1]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, "", true
	case 3:
		catalog := pathSplit[0]
		template := pathSplit[1]
		revisionOrVersion := pathSplit[2]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, revisionOrVersion, true
	default:
		return "", "", "", "", false
	}
}
