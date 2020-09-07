<template>
  <div class="home">
    <HelloWorld msg="Welcome to Your Vue.js App"/>
  </div>
</template>

<script>
// @ is an alias to /src
import HelloWorld from '@/components/Sliders.vue'

export default {
  name: 'Home',
  components: {
    HelloWorld
  }
}
</script>
<style>
  body {
    background-color: #343a40;
  }

</style>