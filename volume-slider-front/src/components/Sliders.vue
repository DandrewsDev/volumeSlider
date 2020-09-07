<template>
  <div>
    <b-container class="main-con">
      <b-row ref="slider-row-ref" class="slider-row">
        <div v-for="item in getAppSliders" :key="item.name">
          <b-col>
            <appSlider :show-remove="showRemove" class="app-slider" :slider="item"></appSlider>
          </b-col>
        </div>
      </b-row>

      <b-row>
        <b-col>
          <b-button v-b-modal.add-slide-modal>Add Slider</b-button>
        </b-col>
        <b-col>
          <b-button v-on:click="removeSliders">Remove Sliders</b-button>
        </b-col>
      </b-row>
    </b-container>

    <b-modal cancel-disabled ok-disabled id="add-slide-modal" title="Add Slider">
      <template v-slot:modal-footer>
        <div></div>
      </template>
      <addSlider></addSlider>
    </b-modal>

  </div>
</template>

<script>
  import appSlider from "./appSlider";
  import addSlider from "./addSlider";
  import {mapActions, mapGetters} from "vuex";
  export default {
    name: 'Sliders',
    components: {
      appSlider,
      addSlider
    },
    data() {
      return {
        mediaItems: [],
        appVolumes: {},
        showRemove: false,
        appSliders: [],
        appList: [

        ],
      }
    },
    mounted() {
      this.updateHostAddress();
      this.fetchAppSliderList();
    },
    methods: {
      ...mapActions([
        'fetchAppSliderList',
        'updateHostAddress',
      ]),
      handleMsg(msg) {
        console.log(msg);
      },
      setAppVolume(appName, newVol) {
        console.log(arguments);
        this.appVolumes[appName] = newVol;
      },
      removeSliders() {
        this.showRemove = !this.showRemove;
      }
    },
    computed:{
      ...mapGetters([
        'getAppSliders'
      ])
    },
    props: {
      msg: String
    }
  }
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
  .app-slider {
    padding-top: 3em;
  }
  .slider-row {
    height: 75%;
  }
</style>
