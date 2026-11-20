package p

func target() {}

func outer() {
	target := func() {}
	mid := func() {
		target := func() {}
		inner := func() { target() }
		inner()
	}
	mid()
}
