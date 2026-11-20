function oldFn() {}
function newFn() {}
function outer() {
  let fn = oldFn;
  function inner() { fn(); }
  fn = newFn;
  inner();
}
