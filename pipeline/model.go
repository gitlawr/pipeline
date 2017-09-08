package pipeline

import (
	"time"

	"github.com/rancher/go-rancher/client"
)

const StepTypeTask = "task"
const StepTypeDeploy = "deploy"
const StepTypeSCM = "scm"
const StepTypeBuild = "build"
const StepTypeService = "service"
const StepTypeUpgradeService = "upgradeService"
const StepTypeUpgradeStack = "upgradeStack"
const StepTypeUpgradeCatalog = "upgradeCatalog"
const (
	ActivityStepWaiting  = "Waiting"
	ActivityStepBuilding = "Building"
	ActivityStepSuccess  = "Success"
	ActivityStepFail     = "Fail"

	ActivityStageWaiting  = "Waiting"
	ActivityStagePending  = "Pending"
	ActivityStageBuilding = "Building"
	ActivityStageSuccess  = "Success"
	ActivityStageFail     = "Fail"
	ActivityStageDenied   = "Denied"

	ActivityWaiting  = "Waiting"
	ActivityPending  = "Pending"
	ActivityBuilding = "Building"
	ActivitySuccess  = "Success"
	ActivityFail     = "Fail"
	ActivityDenied   = "Denied"
)

var PreservedEnvs = [...]string{"CICD_GIT_COMMIT", "CICD_GIT_PREVIOUS_COMMIT", "CICD_GIT_PREVIOUS_SUCCESSFUL_COMMIT",
	"CICD_GIT_BRANCH", "CICD_GIT_LOCAL_BRANCH", "CICD_GIT_URL", "CICD_GIT_COMMITTER_NAME",
	"CICD_GIT_AUTHOR_NAME", "CICD_GIT_COMMITTER_EMAIL", "CICD_GIT_AUTHOR_EMAIL", "CICD_SVN_REVISION",
	"CICD_SVN_URL", "CICD_PIPELINE_NAME", "CICD_PIPELINE_ID", "CICD_TRIGGER_TYPE", "CICD_NODE_NAME", "CICD_ACTIVITY_ID",
	"CICD_ACTIVITY_SEQUENCE",
}

type PipelineSetting struct {
	client.Resource
	HostUrl      string `json:"hostUrl,omitempty" yaml:"hostUrl,omitempty"`
	SMTPHost     string `json:"SMTPHost,omitempty" yaml:"SMTPHost,omitempty"`
	SMTPPort     int    `json:"SMTPPort,omitempty" yaml:"SMTPPort,omitempty"`
	SMTPUsername string `json:"SMTPUsername,omitempty" yaml:"SMTPUsername,omitempty"`
	SMTPPassword string `json:"SMTPPassword,omitempty" yaml:"SMTPPassword,omitempty"`
}

type Pipeline struct {
	client.Resource
	Name            string `json:"name,omitempty" yaml:"name,omitempty"`
	IsActivate      bool   `json:"isActivate,omitempty" yaml:"isActivate,omitempty"`
	VersionSequence string `json:"-" yaml:"-"`
	RunCount        int    `json:"runCount,omitempty" yaml:"runCount,omitempty"`
	LastRunId       string `json:"lastRunId,omitempty" yaml:"lastRunId,omitempty"`
	LastRunStatus   string `json:"lastRunStatus,omitempty" yaml:"lastRunStatus,omitempty"`
	LastRunTime     int64  `json:"lastRunTime,omitempty" yaml:"lastRunTime,omitempty"`
	NextRunTime     int64  `json:"nextRunTime,omitempty" yaml:"nextRunTime,omitempty"`
	CommitInfo      string `json:"commitInfo,omitempty" yaml:"commitInfo,omitempty"`
	Repository      string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch          string `json:"branch,omitempty" yaml:"branch,omitempty"`
	TargetImage     string `json:"targetImage,omitempty" yaml:"target-image,omitempty"`
	File            string `json:"file"`
	WebHookId       int    `json:"webhookId,omitempty" yaml:"webhookId,omitempty"`
	WebHookToken    string `json:"webhookToken,omitempty" yaml:"webhookToken,omitempty"`
	//trigger
	TriggerType     string `json:"triggerType,omitempty" yaml:"triggerType,omitempty"`
	TriggerOnUpdate bool   `json:"triggerOnUpdate,omitempty" yaml:"triggerOnUpdate,omitempty"`
	TriggerSpec     string `json:"triggerSpec" yaml:"triggerSpec,omitempty"`
	TriggerTimezone string `json:"triggerTimezone,omitempty" yaml:"triggerTimezone,omitempty"`

	//email notification settings
	IsEmail         bool   `json:"isEmail,omitempty" yaml:"isEmail,omitempty"`
	IsSendCommitter bool   `json:"isSendCommitter,omitempty" yaml:"isSendCommitter,omitempty"`
	EmailRecipients string `json:"emailRecipients,omitempty" yaml:"emailRecipients,omitempty"`

	Stages []*Stage `json:"stages,omitempty" yaml:"stages,omitempty"`
}

type Stage struct {
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Ordinal     int      `json:"ordinal,omitempty" yaml:"ordinal,omitempty"`
	NeedApprove bool     `json:"needApprove,omitempty" yaml:"need-approve,omitempty"`
	Approvers   []string `json:"approvers,omitempty"`
	Steps       []*Step  `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Step struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Type    string `json:"type,omitempty" yaml:"type,omitempty"`
	Ordinal int    `json:"ordinal,omitempty" yaml:"ordinal,omitempty"`
	//---SCM step
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Webhook    bool   `json:"webhook,omitempty" yaml:"webhook,omitempty"`
	Token      string `json:"token,omitempty" yaml:"token,omitempty"`
	//---Build step
	SourceType     string `json:"sourceType,omitempty" yaml:"sourceType,omitempty"`
	Dockerfile     string `json:"file,omitempty" yaml:"file,omitempty"`
	DockerfilePath string `json:"dockerfilePath,omittempty" yaml:"dockerfilePath,omitempty"`
	TargetImage    string `json:"targetImage,omitempty" yaml:"targetImage,omitempty"`
	PushFlag       bool   `json:"push,omitempty" yaml:"push,omitempty"`
	UserName       string `json:"username,omitempty" yaml:"username,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`

	//---task step
	Command    string       `json:"command,omitempty" yaml:"command,omitempty"`
	Image      string       `json:"image,omitempty" yaml:"image,omitempty"`
	Parameters []string     `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Entrypoint string       `json:"entrypoint,omitempty" yaml:"enrtypoint,omitempty"`
	Args       string       `json:"args,omitempty" yaml:"args,omitempty"`
	Alias      string       `json:"alias,omitempty" yaml:"alias,omitempty"`
	IsService  bool         `json:"isService,omitempty"`
	IsShell    bool         `json:"isShell"`
	Services   []*CIService `json:"services,omitempty"`

	//---upgradeStack step
	//Endpoint,Accesskey,Secretkey
	StackType      string `json:"stackType,omitempty" yaml:"stackType,omitempty"` //catalog or custom
	StackName      string `json:"stackName,omitempty" yaml:"stackType,omitempty"`
	DockerCompose  string `json:"dockerCompose,omitempty" yaml:"docker-compose,omitempty"`
	RancherCompose string `json:"rancherCompose,omitempty" yaml:"rancher-compose,omitempty"`

	//---deploy step
	DeployName        string `json:"deployName,omitempty" yaml:"deploy-name,omitempty"`
	DeployEnvironment string `json:"deployEnvironment,omitempty" yaml:"deploy-environment,omitempty"`
	Count             int    `json:"count,omitempty" yaml:"count,omitempty"`
	//---upgradeService step
	Tag             string            `json:"tag,omitempty" yaml:"tag,omitempty"`
	ServiceSelector map[string]string `json:"serviceSelector,omitempty" yaml:"serviceSelector,omitempty"`
	BatchSize       int               `json:"batchSize,omitempty" yaml:"batchSize,omitempty"`
	Interval        int               `json:"interval,omitempty" yaml:"interval,omitempty"`
	StartFirst      bool              `json:"startFirst,omitempty" yaml:"startFirst,omitempty"`
	DeployEnv       string            `json:"deployEnv,omitempty" yaml:"deployEnv,omitempty"`
	EnvironmentId   string            `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Endpoint        string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Accesskey       string            `json:"accesskey,omitempty" yaml:"accesskey,omitempty"`
	Secretkey       string            `json:"secretkey,omitempty" yaml:"secretkey,omitempty"`

	//---upgradeCatalog step
	//Endpoint,Accesskey,Secretkey,StackName,
	//Repository,Branch,Username,Password,DeployEnv
	DeployFlag bool        `json:"deploy,omitempty" yaml:"deploy,omitempty"`
	ExternalId string      `json:"externalId,omitempty" yaml:"externalId,omitempty"`
	FilesArray []PlainFile `json:"filesAry,omitempty" yaml:"filesAry,omitempty"`
	Readme     string      `json:"readme,omitempty" yaml:"readme,omitempty"`
	Answers    string      `json:"answerString,omitempty" yaml:"answerString,omitempty"`
}

type PlainFile struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Body string `json:"body,omitempty" yaml:"body,omitempty"`
}

type Trigger struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// cron trigger
	Spec     string `json:"spec" yaml:"spec,omitempty"`
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

type BuildStep struct {
	Repository string `json:"-" yaml:"-"`
	Branch     string `json:"-" yaml:"-"`
}

type PipelineProvider interface {
	Init(*Pipeline) error
	RunPipeline(*Pipeline) (*Activity, error)
	RerunActivity(*Activity) error
	RunBuild(*Stage, string) error
	RunStage(*Activity, int) error
	SyncActivity(*Activity) (bool, error)
	GetStepLog(*Activity, int, int) (string, error)
	DeleteFormerBuild(*Activity) error
	OnActivityCompelte(*Activity)
}

type Activity struct {
	client.Resource
	Id              string           `json:"id,omitempty"`
	Pipeline        Pipeline         `json:"pipelineSource,omitempty"`
	PipelineName    string           `json:"pipelineName,omitempty"`
	PipelineVersion string           `json:"pipelineVersion,omitempty"`
	RunSequence     int              `json:"runSequence,omitempty"`
	CommitInfo      string           `json:"commitInfo,omitempty"`
	Status          string           `json:"status,omitempty"`
	PendingStage    int              `json:"pendingStage,omitempty"`
	StartTS         int64            `json:"start_ts,omitempty"`
	StopTS          int64            `json:"stop_ts,omitempty"`
	NodeName        string           `json:"nodename,omitempty"`
	ActivityStages  []*ActivityStage `json:"activity_stages,omitempty"`
}

type ActivityStage struct {
	ActivityId    string          `json:"activity_id,omitempty"`
	Name          string          `json:"name,omitempty"`
	NeedApproval  bool            `json:"need_approval,omitempty"`
	Approvers     []string        `json:"approvers,omitempty"`
	ActivitySteps []*ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64           `json:"start_ts,omitempty"`
	Duration      int64           `json:"duration,omitempty"`
	Status        string          `json:"status,omitempty"`
	RawOutput     string          `json:"rawOutput,omitempty"`
}

type ActivityStep struct {
	Name     string `json:"name,omitempty"`
	Message  string `json:"message,omitempty"`
	Status   string `json:"status,omitempty"`
	StartTS  int64  `json:"start_ts,omitempty"`
	Duration int64  `json:"duration,omitempty"`
}

type CIService struct {
	ContainerName string `json:"containerName,omitempty"`
	Name          string `json:"name,omitempty"`
	Image         string `json:"image,omitempty"`
	Entrypoint    string `json:"entrypoint,omitempty"`
	Command       string `json:"command,omitempty"`
}

func ToDemoActivity() *Activity {
	startTS := (time.Now().Unix() - 30) * 1000
	stopTS := time.Now().Unix()
	r := Activity{
		Id:              "test",
		PipelineName:    "test1",
		PipelineVersion: "0",
		Status:          ActivitySuccess,
		StartTS:         startTS,
		StopTS:          stopTS,
		ActivityStages: []*ActivityStage{
			&ActivityStage{
				Name:         "build",
				NeedApproval: false,
				StartTS:      startTS,
				Status:       ActivityStageSuccess,
				RawOutput:    "",
				ActivitySteps: []*ActivityStep{
					&ActivityStep{
						Name:    "build",
						Message: "",
						Status:  ActivityStageSuccess,
					},
				},
			},
		},
	}
	return &r
}
