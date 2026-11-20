package p

type I interface{ M() }

func run(x I) {
	x.M()
}
