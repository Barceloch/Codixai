package p

func Helper() {}

func F() {
	Helper := func() {}
	Helper()
}
