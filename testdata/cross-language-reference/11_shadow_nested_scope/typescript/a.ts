function helper(): void {}
function outer(): void {
  const helper = (): void => {};
  function inner(): void {
    const helper = (): void => {};
    helper();
  }
  helper();
}
