def old_fn():
    pass

def new_fn():
    pass

def outer():
    fn = old_fn
    def inner():
        fn()
    fn = new_fn
    inner()
