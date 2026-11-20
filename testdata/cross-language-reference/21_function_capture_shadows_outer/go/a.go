package p

func target() {}

func outer() {
	inner := func() {
		target := func() {}
		target()
	}
	inner()
}
