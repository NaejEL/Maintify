<template>
  <div id="app">
    <h1>🔧 Maintify - Test Simple</h1>
    <p>Application Vue fonctionnelle !</p>
    <p>{{ message }}</p>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      message: 'Hello from Vue.js!'
    }
  }
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}
</style>
