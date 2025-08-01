<template>
  <div class="simple-test">
    <h1>Test Page - Simple</h1>
    <p>Cette page de test fonctionne</p>
    <router-link to="/login">Aller à Login</router-link>
  </div>
</template>

<script>
export default {
  name: 'SimpleTest'
}
</script>

<style scoped>
.simple-test {
  padding: 20px;
  text-align: center;
}
</style>
