function target(): void {}
function outer(): void {
  const target = (): void => {};
  function inner(): void { target(); }
  inner();
}
