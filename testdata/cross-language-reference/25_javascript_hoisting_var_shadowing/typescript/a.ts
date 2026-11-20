function target(): void {}
function outer(): void {
  function inner(): void { target(); }
  var target = (): void => {};
  inner();
}
