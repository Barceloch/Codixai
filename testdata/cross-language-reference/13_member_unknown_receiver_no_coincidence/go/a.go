package p

type UserService struct{}

func (UserService) Create() {}

func run(svc interface{}) {
	svc.Create()
}
