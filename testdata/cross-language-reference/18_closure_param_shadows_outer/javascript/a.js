function helper() {}
function outer() {
  function inner(helper) {
    helper();
  }
}
