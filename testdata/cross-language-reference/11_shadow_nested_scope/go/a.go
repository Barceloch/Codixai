package p

func helper() {}

func outer() {
	helper := func() {}
	{
		helper := func() {}
		helper()
	}
	helper()
}
