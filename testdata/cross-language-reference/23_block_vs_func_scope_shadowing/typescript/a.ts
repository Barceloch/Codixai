function target(): void {}
function outer(): void {
  {
    let target = (): void => {};
    target();
  }
}
