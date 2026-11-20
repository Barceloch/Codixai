function target(): void {}
function outer(): void {
  function inner(): void {
    const target = (): void => {};
    target();
  }
  inner();
}
