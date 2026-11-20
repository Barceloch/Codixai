package p

type I interface{ M() }

func f(x I) { x.M() }
