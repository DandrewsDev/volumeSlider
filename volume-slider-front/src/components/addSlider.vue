<template>
    <div>
        <b-form @submit="onSubmit" v-if="show">
            <b-form-group
                    id="input-group-1"
                    label="Application name:"
                    label-for="input-1"
                    description="This is the name that will display below the slider (Ex: Firefox)">
                <b-form-input
                        id="input-1"
                        v-model="form.name"
                        type="text"
                        required
                        placeholder="App name"
                ></b-form-input>
            </b-form-group>

            <b-form-group id="input-group-2" label="Exe Filename:" label-for="input-2"
                          description="Name of .exe to control (Ex: firefox.exe)">
                <b-form-input
                        id="input-2"
                        v-model="form.exeName"
                        required
                        placeholder="Exe name"
                ></b-form-input>
            </b-form-group>

            <b-button type="submit" variant="primary">Submit</b-button>
        </b-form>
    </div>
</template>

<script>
    import {mapActions, mapGetters} from "vuex";

    export default {
        name: "addSlider",
        data() {
            return {
                form: {
                    exeName: '',
                    name: '',
                },
                show: true
            }
        },
        computed: {
          ...mapGetters([
              'getAppSliders'
          ])
        },
        methods: {
            ...mapActions([
                'updateAppSliderList',
                'postAppSliderList'
            ]),
            onSubmit(evt) {
                evt.preventDefault()
                let sliders = this.getAppSliders;
                sliders.push({appName: this.form.name, exeName: this.form.exeName});
                this.updateAppSliderList(sliders);
                this.postAppSliderList();
                this.$bvModal.hide("add-slide-modal");
            },
        }
    }
</script>

<style scoped>

</style>