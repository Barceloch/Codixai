def target():
    pass

def outer():
    if True:
        target = lambda: None
        target()
