package p

func helper() {}

func outer() {
	f := func(helper func()) {
		helper()
	}
	_ = f
}
