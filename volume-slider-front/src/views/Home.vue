<template>
  <div class="home">
    <Sliders/>
  </div>
</template>

<script>
// @ is an alias to /src
import Sliders from '@/components/Sliders.vue'

export default {
  name: 'Home',
  components: {
    Sliders
  }
}
</script>
<style>
  body {
    background-color: #343a40;
  }

</style>