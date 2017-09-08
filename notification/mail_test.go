package notification

import "testing"

func TestSend(t *testing.T) {
	notifier := NewEmailNotifier("smtp.qq.com", 587, "lawrlee@foxmail.com", "test123456")
	err := notifier.SendMail("lawrleegle@gmail.com,153319347@qq.com", "subject", "body")
	if err != nil {
		t.Error(err)
	}
}
