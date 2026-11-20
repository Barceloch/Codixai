package p

func oldFn() {}
func newFn() {}

func outer() {
	fn := oldFn
	inner := func() { fn() }
	fn = newFn
	inner()
}
