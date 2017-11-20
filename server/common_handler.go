package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/pipeline/model"
	"github.com/rancher/pipeline/server/service"
	"github.com/rancher/pipeline/server/webhook"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

func (s *Server) Webhook(rw http.ResponseWriter, req *http.Request) error {
	var signature string
	var event_type string
	logrus.Debugln("get webhook request")
	logrus.Debugf("get header:%v", req.Header)
	logrus.Debugf("get url:%v", req.RequestURI)

	if signature = req.Header.Get("X-Hub-Signature"); len(signature) == 0 {
		return errors.New("No signature!")
	}
	if event_type = req.Header.Get("X-GitHub-Event"); len(event_type) == 0 {
		return errors.New("No event!")
	}

	if event_type == "ping" {
		rw.Write([]byte("pong"))
		return nil
	}
	if event_type != "push" {
		logrus.Errorf("not push event")
		return errors.New("not push event")
	}

	id := req.FormValue("pipelineId")
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	r := service.GetPipelineById(id)
	if r == nil {
		err := errors.Wrapf(model.ErrPipelineNotFound, "pipeline <%s>", id)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("pipeline not found!"))
		return err
	}
	logrus.Debugf("webhook trigger,id:%v,event:%v,signature:%v,body:\n%v\n%v", id, event_type, signature, body, string(body))
	if !webhook.VerifyGithubWebhookSignature([]byte(r.WebHookToken), signature, body) {
		return errors.New("Invalid signature")
	}
	logrus.Debugf("token validate pass")

	//check branch
	payload := &github.WebHookPayload{}
	if err := json.Unmarshal(body, payload); err != nil {
		return err
	}
	if *payload.Ref != "refs/heads/"+r.Stages[0].Steps[0].Branch {
		logrus.Warningf("branch not match:%v,%v", *payload.Ref, r.Stages[0].Steps[0].Branch)
		return nil
	}

	if !r.IsActivate {
		logrus.Errorf("pipeline is not activated!")
		return errors.New("pipeline is not activated!")
	}
	if _, err = service.RunPipeline(s.Provider, id, model.TriggerTypeWebhook); err != nil {
		rw.Write([]byte("run pipeline error!"))
		return err
	}
	rw.Write([]byte("run pipeline success!"))
	logrus.Infof("webhook run success")
	return nil
}

func (s *Server) ServeStatusWS(w http.ResponseWriter, r *http.Request) error {
	apiContext := api.GetApiContext(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logrus.Errorf("ws handshake error")
		}
		return err
	}
	uid, err := util.GetCurrentUser(r.Cookies())
	//logrus.Infof("got currentUser,%v,%v", uid, err)
	if err != nil || uid == "" {
		logrus.Errorf("get currentUser fail,%v,%v", uid, err)
	}
	connHolder := &ConnHolder{agent: MyAgent, conn: conn, send: make(chan WSMsg)}

	connHolder.agent.register <- connHolder

	//new go routines
	go connHolder.DoWrite(apiContext, uid)
	connHolder.DoRead()

	return nil
}

//list available env vars
func (s *Server) ListEnvVars(rw http.ResponseWriter, req *http.Request) error {
	b, err := json.Marshal(model.PreservedEnvs)
	_, err = rw.Write(b)
	return err
}

func (s *Server) StepStart(rw http.ResponseWriter, req *http.Request) error {
	v := req.URL.Query()
	activityId := v.Get("id")
	stageOrdinal, err := strconv.Atoi(v.Get("stageOrdinal"))
	if err != nil {
		return err
	}
	stepOrdinal, err := strconv.Atoi(v.Get("stepOrdinal"))
	if err != nil {
		return err
	}

	mutex := MyAgent.getActivityLock(activityId)
	mutex.Lock()
	defer mutex.Unlock()

	logrus.Debugf("get stepstart event,paras:%v,%v,%v", activityId, stageOrdinal, stepOrdinal)
	activity, err := service.GetActivity(activityId)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	service.StartStep(activity, stageOrdinal, stepOrdinal)
	if err = service.UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         activity,
	}
	return nil
}

func (s *Server) StepFinish(rw http.ResponseWriter, req *http.Request) error {
	//get activityId,stageOrdinal,stepOrdinal from request
	v := req.URL.Query()
	activityId := v.Get("id")
	status := v.Get("status")
	stageOrdinal, err := strconv.Atoi(v.Get("stageOrdinal"))
	if err != nil {
		return err
	}
	stepOrdinal, err := strconv.Atoi(v.Get("stepOrdinal"))
	if err != nil {
		return err
	}
	mutex := MyAgent.getActivityLock(activityId)
	mutex.Lock()
	defer mutex.Unlock()

	logrus.Debugf("get stepfinish event,paras:%v,%v,%v", activityId, stageOrdinal, stepOrdinal)
	activity, err := service.GetActivity(activityId)
	if err != nil {
		return err
	}
	if stageOrdinal < 0 || stepOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return errors.New("step index invalid")
	}
	if status == "SUCCESS" {
		service.SuccessStep(activity, stageOrdinal, stepOrdinal)
		service.Triggernext(activity, stageOrdinal, stepOrdinal, s.Provider)
	} else if status == "FAILURE" {
		service.FailStep(activity, stageOrdinal, stepOrdinal)
	}

	//update commitinfo for SCM step
	if stageOrdinal == 0 && stepOrdinal == 0 {
		activity.CommitInfo = req.FormValue("GIT_COMMIT")
		activity.EnvVars["CICD_GIT_COMMIT"] = activity.CommitInfo
	}

	if err = service.UpdateActivity(activity); err != nil {
		return err
	}

	MyAgent.broadcast <- WSMsg{
		Id:           uuid.Rand().Hex(),
		Name:         "resource.change",
		ResourceType: "activity",
		Time:         time.Now(),
		Data:         activity,
	}
	s.UpdateLastActivity(activity)

	if activity.Status == model.ActivityFail || activity.Status == model.ActivitySuccess {
		s.Provider.OnActivityCompelte(activity)
	}

	return nil
}
