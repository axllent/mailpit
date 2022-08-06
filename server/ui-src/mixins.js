import axios from 'axios'

// FakeModal is used to return a fake Bootstrap modal
// if the ID returns nothing
function FakeModal() { }
FakeModal.prototype.hide = function () { alert('close fake modal') }
FakeModal.prototype.show = function () { alert('open fake modal') }

/* Common mixin functions used in apps */
const commonMixins = {
    data() {
        return {
            loading: 0,
        }
    },

    methods: {
        getFileSize: function (bytes) {
            var i = Math.floor(Math.log(bytes) / Math.log(1024));
            return (bytes / Math.pow(1024, i)).toFixed(1) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
        },

        formatNumber: function (nr) {
            return new Intl.NumberFormat().format(nr);
        },

        // Ajax error message
        handleError: function (error) {
            // handle error
            if (error.response && error.response.data) {
                // The request was made and the server responded with a status code
                // that falls out of the range of 2xx
                if (error.response.data.Error) {
                    alert(error.response.data.Error)
                } else {
                    alert(error.response.data);
                }
            } else if (error.request) {
                // The request was made but no response was received
                // `error.request` is an instance of XMLHttpRequest in the browser and an instance of
                // http.ClientRequest in node.js
                alert('Error sending data to the server. Please try again.');
            } else {
                // Something happened in setting up the request that triggered an Error
                alert(error.message);
            }
        },

        // generic modal get/set function
        modal: function (id) {
            let e = document.getElementById(id);
            if (e) {
                return bootstrap.Modal.getOrCreateInstance(e);
            }
            // in case there are open/close actions
            return new FakeModal();
        },

        // generic modal get/set function
        offcanvas: function (id) {
            var e = document.getElementById(id);
            if (e) {
                return bootstrap.Offcanvas.getOrCreateInstance(e);
            }
            // in case there are open/close actions
            return new FakeModal();
        },

        /**
         * Axios GET request
         *
         * @params string   url
         * @params array    array parameters Object/array
         * @params function callback function
         */
        get: function (url, values, callback) {
            let self = this;
            self.loading++;
            axios.get(url, { params: values })
                .then(callback)
                .catch(self.handleError)
                .then(function () {
                    // always executed
                    if (self.loading > 0) {
                        self.loading--;
                    }
                });
        },

        /**
         * Axios Post request
         *
         * @params string   url
         * @params array    array parameters Object/array
         * @params function callback function
         */
        post: function (url, values, callback) {
            let self = this;
            const params = new URLSearchParams();
            for (const [key, value] of Object.entries(values)) {
                params.append(key, value);
            }
            self.loading++;
            axios.post(url, params)
                .then(callback)
                .catch(self.handleError)
                .then(function () {
                    // always executed
                    if (self.loading > 0) {
                        self.loading--;
                    }
                });
        },

        /**
         * Axios DELETE request (REST only)
         *
         * @params string   url
         * @params array    array parameters Object/array
         * @params function callback function
         */
        delete: function (url, values, callback) {
            let self = this;
            self.loading++;
            axios.delete(url, { data: values })
                .then(callback)
                .catch(self.handleError)
                .then(function () {
                    // always executed
                    if (self.loading > 0) {
                        self.loading--;
                    }
                });
        }
    }
}


export default commonMixins 
