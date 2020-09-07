import Vue from 'vue'
import Vuex from 'vuex'
import axios from 'axios';

Vue.use(Vuex)

export default new Vuex.Store({
  state: {
    hostAddress: "",
    appNames: [
    ]
  },
  getters: {
    getAppSliders: (state) => state.appNames,
  },
  mutations: {
    setAppSliders: (state, nameList) => (state.appNames = nameList),
    setHostAddress: (state, add) => (state.hostAddress = add),
  },
  actions: {
    updateHostAddress({commit}) {
      if (window.location.href.includes('file:///')) {
        commit('setHostAddress', 'http://localhost:1323')
      }
    },
    fetchAppSliderList({commit, state}) {
      axios.get(state.hostAddress + '/slider-list')
          .then(response => {
            commit('setAppSliders', response.data)
          })
          .catch(e => {
            console.log(e);
            this.disabled = true;
          })
    },
    postAppSliderList({state}) {
      axios.post(state.hostAddress + '/slider-list', {
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
