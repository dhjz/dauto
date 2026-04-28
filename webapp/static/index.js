const { createApp } = Vue
var app = createApp({
  data() {
    return {
      message: 'Hello Vue!'
    }
  }
})
app.mount('#app')