<template>
    <div class="app-slider-base">

        <b-button v-on:click="removeSlider(slider.appName)" v-if="showRemove" size="md" class="mb-2">
            <b-icon variant="light"  icon="x-square-fill"></b-icon>
        </b-button>
        <p></p>
        <vue-slider
                :disabled="disabled"
                :id="sliderId"
                @change="setAppVolume"
                direction="btt"
                :height="300"
                :width="30"
                :dotSize="32"
                tooltip="always"
                tooltipPlacement="bottom"
                :process-style="{ backgroundColor: sliderColor }"
                :tooltip-style="{ backgroundColor: sliderColor, borderColor: sliderColor }"

                v-model="value" />
        <p></p>
        <label v-bind:class="{ 'slider-label': !disabled }" :for="sliderId">{{slider.appName}}</label>
    </div>
</template>

<script>
    import axios from 'axios';
    import VueSlider from 'vue-slider-component'
    import 'vue-slider-component/theme/antd.css'
    import {mapActions, mapGetters} from "vuex";

    export default {
        name: "appSlider",
        data() {
            return {
                value: 40,
                disabled: false,
            }
        },
        components: {
            VueSlider
        },
        props: {
            slider: Object,
            showRemove: {
                type: Boolean,
                default: false
            },
        },
        computed: {
            ...mapGetters([
                'getAppSliders'
            ]),
            sliderId () {
                let st = "slider-vol-" + this.slider.appName;
                st.replace(/[^a-z0-9]/gi, '_').toLowerCase();
                return st
            },
            sliderColor () {
                if (!this.disabled) {
                    return '#688cb0'
                }
                return '#000000'
            },
        },
        mounted() {
            this.updateSliderData();
        },
        watch: {
            slider: function (newSliderInfo) {
                this.updateSliderData();
                console.log(newSliderInfo);
                console.log('newSliderInfo^^');
            }
        },
        methods: {
            ...mapActions([
                'updateAppSliderList',
                'postAppSliderList'
            ]),
            updateSliderData() {
                axios.get( '/app-vol/' + this.slider.exeName)
                        .then(response => {
                            this.value = response.data.volume;
                            this.disabled = false;
                        })
                        .catch(e => {
                            console.log(e);
                            this.disabled = true;
                        })
            },
            removeSlider(slider) {
                let sliders = this.getAppSliders;
                for (let i=0; i < sliders.length; i++) {
                    if (sliders[i].appName === slider) {
                        sliders.splice(i, 1);
                        this.updateAppSliderList(sliders);
                        this.postAppSliderList();
                    }
                }
            },
            setAppVolume(newVol) {
                axios.post('/app-vol-set', {
                    name: this.slider.exeName,
                    volume: parseInt(newVol)
                })
            },
        },
    }
</script>

<style scoped>
    .slider-remove-icon {
        padding-right: .5em;
    }
    .slider-label {
        color: #688cb0;
        font-weight: 600;
    }
    .app-slider-base{
        display: inline-grid;
    }
</style>