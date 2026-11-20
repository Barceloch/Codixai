package p

func target() {}

func outer() {
	{
		target := func() {}
		target()
	}
}
