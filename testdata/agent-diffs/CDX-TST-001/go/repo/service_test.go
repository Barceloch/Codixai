package service

import "testing"

type Service struct {
	Result string
}

func (s *Service) Call() {
	s.Result = "called"
}

func TestService(t *testing.T) {
	s := &Service{}
	s.Call()
}

func TestService_Call(t *testing.T) {
	s := &Service{}
	s.Call()
}
