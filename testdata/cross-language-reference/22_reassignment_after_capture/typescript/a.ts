function oldFn(): void {}
function newFn(): void {}
function outer(): void {
  let fn = oldFn;
  function inner(): void { fn(); }
  fn = newFn;
  inner();
}
