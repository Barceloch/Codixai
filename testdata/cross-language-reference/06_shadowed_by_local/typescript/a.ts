function helper(): void {}
function caller(helper: () => void): void {
  helper();
}
