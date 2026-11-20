function helper(): void {}
function outer(): void {
  function inner(helper: () => void): void {
    helper();
  }
}
