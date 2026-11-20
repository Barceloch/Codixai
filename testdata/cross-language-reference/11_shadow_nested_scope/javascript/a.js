function helper() {}
function outer() {
  const helper = () => {};
  function inner() {
    const helper = () => {};
    helper();
  }
  helper();
}
