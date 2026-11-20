function target() {}
function outer() {
  {
    target();
    let target = () => {};
  }
}
