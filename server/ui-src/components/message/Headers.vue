<script>
import commonMixins from '../../mixins/CommonMixins'

export default {
    props: {
        message: Object
    },

    mixins: [commonMixins],

    data() {
        return {
            headers: false
        }
    },

    mounted() {
        let uri = this.resolve('/api/v1/message/' + this.message.ID + '/headers')
        this.get(uri, false, (response) => {
            this.headers = response.data
        });
    },

}
</script>

<template>
    <div v-if="headers" class="small">
        <div v-for="values, k in headers" class="row mb-2 pb-2 border-bottom w-100">
            <div class="col-md-4 col-lg-3 col-xl-2 mb-2"><b>{{ k }}</b></div>
            <div class="col-md-8 col-lg-9 col-xl-10 text-body-secondary">
                <div v-for="x in values" class="mb-2 text-break">{{ x }}</div>
            </div>
        </div>
    </div>
</template>
