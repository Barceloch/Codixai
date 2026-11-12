package p

func Do() {}

type T struct{}

func (T) Do() {}

func F(x T) { x.Do() }
