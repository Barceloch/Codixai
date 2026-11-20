function target(): void {}
function outer(): void {
  {
    target();
    let target = (): void => {};
  }
}
