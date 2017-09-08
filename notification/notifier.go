package notification

type notifier interface {
	Notify(recepients string, message string) error
}
