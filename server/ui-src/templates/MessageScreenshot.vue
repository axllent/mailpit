
<script>
import { domToPng } from 'modern-screenshot'

export default {
    props: {
        message: Object,
    },

    data() {
        return {
            html: false,
            loading: false
        }
    },

    methods: {
        initScreenshot: function () {
            this.loading = true
            // remove base tag, if set
            let h = this.message.HTML.replace(/<base .*>/mi, '')

            // Outlook hacks - else screenshot returns blank image
            h = h.replace(/<html [^>]+>/mgi, '<html>') // remove html attributes
            h = h.replace(/<o:p><\/o:p>/mg, '') // remove empty `<o:p></o:p>` tags
            h = h.replace(/<o:/mg, '<') // replace `<o:p>` tags with `<p>` 
            h = h.replace(/<\/o:/mg, '</') // replace `</o:p>` tags with `</p>` 

            // update any inline `url(...)` absolute links
            const urlRegex = /(url\((\'|\")?(https?:\/\/[^\)\'\"]+)(\'|\")?\))/mgi;
            h = h.replaceAll(urlRegex, function (match, p1, p2, p3) {
                if (typeof p2 === 'string') {
                    return `url(${p2}proxy?url=` + encodeURIComponent(p3) + `${p2})`
                }
                return `url(proxy?url=` + encodeURIComponent(p3) + `)`
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
                    i.setAttribute('href', 'proxy?url=' + encodeURIComponent(src))
                }
            }

            // replace images with proxy links
            let images = doc.getElementsByTagName('img')
            for (let i of images) {
                let src = i.getAttribute('src')
                if (src && src.match(/^https?:\/\//i) && src.indexOf(window.location.origin + window.location.pathname) !== 0) {
                    i.setAttribute('src', 'proxy?url=' + encodeURIComponent(src))
                }
            }

            // replace background="" attributes with proxy links
            let backgrounds = doc.querySelectorAll("[background]")
            for (let i of backgrounds) {
                let src = i.getAttribute('background')

                if (src && src.match(/^https?:\/\//i) && src.indexOf(window.location.origin + window.location.pathname) !== 0) {
                    // replace with proxy link
                    i.setAttribute('background', 'proxy?url=' + encodeURIComponent(src))
                }
            }

            // set html with manipulated document content
            this.html = new XMLSerializer().serializeToString(doc)
        },

        doScreenshot: function () {
            let self = this

            let width = document.getElementById('message-view').getBoundingClientRect().width

            let prev = document.getElementById('preview-html')
            if (prev && prev.getBoundingClientRect().width) {
                width = prev.getBoundingClientRect().width
            }

            if (width < 300) {
                width = 300
            }

            let i = document.getElementById('screenshot-html')

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
                link.download = self.message.ID + '.png'
                link.href = dataUrl
                link.click()
                self.loading = false
                self.html = false
            })
        }
    }
}
</script>

<template>
    <iframe v-if="html" :srcdoc="html" v-on:load="doScreenshot" frameborder="0" id="screenshot-html"
        style="position: absolute; margin-left: -100000px;">
    </iframe>

    <div id="loading" v-if="loading">
        <div class="d-flex justify-content-center align-items-center h-100">
            <div class="spinner-border text-secondary" role="status">
                <span class="visually-hidden">Loading...</span>
            </div>
        </div>
    </div>
</template>
