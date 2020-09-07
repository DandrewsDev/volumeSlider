import Vue from 'vue'
import Vuex from 'vuex'
import axios from 'axios';

Vue.use(Vuex)

export default new Vuex.Store({
  state: {
    appNames: [
    ]
  },
  getters: {
    getAppSliders: (state) => state.appNames,
  },
  mutations: {
    setAppSliders: (state, nameList) => (state.appNames = nameList),
  },
  actions: {
    fetchAppSliderList({commit}) {
      axios.get( '/slider-list')
          .then(response => {
            commit('setAppSliders', response.data)
          })
          .catch(e => {
            console.log(e);
            this.disabled = true;
          })
    },
    postAppSliderList({state}) {
      axios.post( '/slider-list', {
        appNames: state.appNames
      }).then(r => console.log(r))
    },
    updateAppSliderList({commit}, value) {
      commit('setAppSliders', value)
    }
  },
  modules: {
  }
})
