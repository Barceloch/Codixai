package p

func target() {}

func outer() {
	target := func() {}
	inner := func() { target() }
	inner()
}
