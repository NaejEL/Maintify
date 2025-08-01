<template>
  <div id="app">
    <h1>🔧 Maintify - Version Progressive</h1>
    <p>Application de base sans router ni store</p>
    <div class="status">
      <p>✅ Vue.js fonctionnel</p>
      <p>✅ Store : ajouté</p>
      <p>✅ Router : ajouté</p>
      <p>✅ i18n : ajouté</p>
    </div>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
#app {
  font-family: Arial, sans-serif;
  text-align: center;
  margin-top: 50px;
}

.status {
  background: #f5f5f5;
  padding: 20px;
  margin: 20px auto;
  max-width: 400px;
  border-radius: 5px;
}

.status p {
  margin: 10px 0;
}
</style>
