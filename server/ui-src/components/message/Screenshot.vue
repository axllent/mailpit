<script>
import AjaxLoader from '../AjaxLoader.vue'
import CommonMixins from '../../mixins/CommonMixins'
import { domToPng } from 'modern-screenshot'

export default {
    props: {
        message: Object,
    },

    mixins: [CommonMixins],

    components: {
        AjaxLoader,
    },

    data() {
        return {
            html: false,
            loading: 0
        }
    },

    methods: {
        initScreenshot() {
            this.loading = 1
            // remove base tag, if set
            let h = this.message.HTML.replace(/<base .*>/mi, '')
            let proxy = this.resolve('/proxy')

            // Outlook hacks - else screenshot returns blank image
            h = h.replace(/<html [^>]+>/mgi, '<html>') // remove html attributes
            h = h.replace(/<o:p><\/o:p>/mg, '') // remove empty `<o:p></o:p>` tags
            h = h.replace(/<o:/mg, '<') // replace `<o:p>` tags with `<p>` 
            h = h.replace(/<\/o:/mg, '</') // replace `</o:p>` tags with `</p>` 

            // update any inline `url(...)` absolute links
            const urlRegex = /(url\((\'|\")?(https?:\/\/[^\)\'\"]+)(\'|\")?\))/mgi;
            h = h.replaceAll(urlRegex, (match, p1, p2, p3) => {
                if (typeof p2 === 'string') {
                    return `url(${p2}${proxy}?url=` + encodeURIComponent(this.decodeEntities(p3)) + `${p2})`
                }
                return `url(${proxy}?url=` + encodeURIComponent(this.decodeEntities(p3)) + `)`
            })

            // create temporary document to manipulate
            let doc = document.implementation.createHTMLDocument();
            doc.open()
            doc.write(h)
            doc.close()

            // remove any <script> tags
            let scripts = doc.getElementsByTagName('script')
            for (let i of scripts) {
                i.parentNode.removeChild(i)
            }

            // replace stylesheet links with proxy links
            let stylesheets = doc.getElementsByTagName('link')
            for (let i of stylesheets) {
                let src = i.getAttribute('href')

                if (src && src.match(/^https?:\/\//i) && src.indexOf(window.location.origin + window.location.pathname) !== 0) {
                    i.setAttribute('href', `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)))
                }
            }

            // replace images with proxy links
            let images = doc.getElementsByTagName('img')
            for (let i of images) {
                let src = i.getAttribute('src')
                if (src && src.match(/^https?:\/\//i) && src.indexOf(window.location.origin + window.location.pathname) !== 0) {
                    i.setAttribute('src', `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)))
                }
            }

            // replace background="" attributes with proxy links
            let backgrounds = doc.querySelectorAll("[background]")
            for (let i of backgrounds) {
                let src = i.getAttribute('background')

                if (src && src.match(/^https?:\/\//i) && src.indexOf(window.location.origin + window.location.pathname) !== 0) {
                    // replace with proxy link
                    i.setAttribute('background', `${proxy}?url=` + encodeURIComponent(this.decodeEntities(src)))
                }
            }

            // set html with manipulated document content
            this.html = new XMLSerializer().serializeToString(doc)
        },

        // HTML decode function
        decodeEntities(s) {
            let e = document.createElement('div')
            e.innerHTML = s
            let str = e.textContent
            e.textContent = ''
            return str
        },

        doScreenshot() {
            let width = document.getElementById('message-view').getBoundingClientRect().width

            let prev = document.getElementById('preview-html')
            if (prev && prev.getBoundingClientRect().width) {
                width = prev.getBoundingClientRect().width
            }

            if (width < 300) {
                width = 300
            }

            const i = document.getElementById('screenshot-html')

            // set the iframe width
            i.style.width = width + 'px'

            let body = i.contentWindow.document.querySelector('body')

            // take screenshot of iframe
            domToPng(body, {
                backgroundColor: '#ffffff',
                height: i.contentWindow.document.body.scrollHeight + 20,
                width: width,
            }).then(dataUrl => {
                const link = document.createElement('a')
                link.download = this.message.ID + '.png'
                link.href = dataUrl
                link.click()
                this.loading = 0
                this.html = false
            })
        }
    }
}
</script>

<template>
    <iframe v-if="html" :srcdoc="html" v-on:load="doScreenshot" frameborder="0" id="screenshot-html"
        style="position: absolute; margin-left: -100000px;">
    </iframe>

    <AjaxLoader :loading="loading" />
</template>
