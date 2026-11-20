function target() {}
function outer() {
  {
    let target = () => {};
    target();
  }
}
