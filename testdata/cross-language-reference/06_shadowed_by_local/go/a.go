package p

func Helper() {}

func f() {
	Helper := func() {}
	Helper()
}
