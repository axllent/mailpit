package htmlcheck

import "regexp"

// HTML tests
var htmlTests = map[string]string{
	// body check is manually done because it always exists in *goquery.Document
	"html-body": "body",
	// HTML tests
	"html-object":         "object, embed, image, pdf",
	"html-link":           "link",
	"html-hr":             "hr",
	"html-dialog":         "dialog",
	"html-srcset":         "[srcset]",
	"html-picture":        "picture",
	"html-svg":            "svg",
	"html-progress":       "progress",
	"html-required":       "[required]",
	"html-meter":          "meter",
	"html-audio":          "audio",
	"html-form":           "form",
	"html-input-submit":   "submit",
	"html-button-reset":   "button[type=\"reset\"]",
	"html-button-submit":  "submit, button[type=\"submit\"]",
	"html-base":           "base",
	"html-input-checkbox": "checkbox",
	"html-input-hidden":   "[type=\"hidden\"]",
	"html-input-radio":    "radio",
	"html-input-text":     "input[type=\"text\"]",
	"html-video":          "video",
	"html-semantics":      "article, aside, details, figcaption, figure, footer, header, main, mark, nav, section, summary, time",
	"html-select":         "select",
	"html-textarea":       "textarea",
	"html-anchor-links":   "a[href^=\"#\"]",
	"html-style":          "style",
	"html-image-maps":     "map, img[usemap]",
}

// Image tests using regex to match against img[src]
var imageRegexpTests = map[string]*regexp.Regexp{
	"image-apng":   regexp.MustCompile(`(?i)\.apng$`),       // 78.723404
	"image-avif":   regexp.MustCompile(`(?i)\.avif$`),       // 14.864864
	"image-base64": regexp.MustCompile(`^(?i)data:image\/`), // 61.702126
	"image-bmp":    regexp.MustCompile(`(?i)\.bmp$`),        // 89.3617
	"image-gif":    regexp.MustCompile(`(?i)\.gif$`),        // 89.3617
	"image-hdr":    regexp.MustCompile(`(?i)\.hdr$`),        // 12.5
	"image-heif":   regexp.MustCompile(`(?i)\.heif$`),       // 0
	"image-ico":    regexp.MustCompile(`(?i)\.ico$`),        // 87.23404
	"image-mp4":    regexp.MustCompile(`(?i)\.mp4$`),        // 26.53061
	"image-ppm":    regexp.MustCompile(`(?i)\.ppm$`),        // 2.0833282
	"image-svg":    regexp.MustCompile(`(?i)\.svg$`),        // 64.91228
	"image-tiff":   regexp.MustCompile(`(?i)\.tiff?$`),      // 38.29787
	"image-webp":   regexp.MustCompile(`(?i)\.webp$`),       // 59.649124
}

var cssInlineTests = map[string]string{
	"css-accent-color":                   "[style*=\"accent-color:\"]",                                           // 6.6666718
	"css-align-items":                    "[style*=\"align-items:\"]",                                            // 60.784313
	"css-aspect-ratio":                   "[style*=\"aspect-ratio:\"]",                                           // 30
	"css-background-blend-mode":          "[style*=\"background-blend-mode:\"]",                                  // 61.70213
	"css-background-clip":                "[style*=\"background-clip:\"]",                                        // 61.70213
	"css-background-color":               "[style*=\"background-color:\"], [bgcolor]",                            // 90
	"css-background-image":               "[style*=\"background-image:\"]",                                       // 57.62712
	"css-background-origin":              "[style*=\"background-origin:\"]",                                      // 61.70213
	"css-background-position":            "[style*=\"background-position:\"]",                                    // 61.224487
	"css-background-repeat":              "[style*=\"background-repeat:\"]",                                      // 67.34694
	"css-background-size":                "[style*=\"background-size:\"]",                                        // 61.702126
	"css-background":                     "[style*=\"background:\"], [background]",                               // 57.407406
	"css-block-inline-size":              "[style*=\"block-inline-size:\"]",                                      // 46.93877
	"css-border-image":                   "[style*=\"border-image:\"]",                                           // 52.173912
	"css-border-inline-block-individual": "[style*=\"border-inline:\"]",                                          // 18.518517
	"css-border-radius":                  "[style*=\"border-radius:\"]",                                          // 67.34694
	"css-border":                         "[style*=\"border:\"], [border]",                                       // 86.95652
	"css-box-shadow":                     "[style*=\"box-shadow:\"]",                                             // 43.103447
	"css-box-sizing":                     "[style*=\"box-sizing:\"]",                                             // 71.739136
	"css-caption-side":                   "[style*=\"caption-side:\"]",                                           // 84
	"css-clip-path":                      "[style*=\"clip-path:\"]",                                              // 43.396225
	"css-column-count":                   "[style*=\"column-count:\"]",                                           // 67.391304
	"css-column-layout-properties":       "[style*=\"column-layout-properties:\"]",                               // 47.368423
	"css-conic-gradient":                 "[style*=\"conic-gradient:\"]",                                         // 38.461536
	"css-direction":                      "[style*=\"direction:\"]",                                              // 97.77778
	"css-display-flex":                   "[style*=\"display:flex\"]",                                            // 53.448277
	"css-display-grid":                   "[style*=\"display:grid\"]",                                            // 54.347824
	"css-display-none":                   "[style*=\"display:none\"]",                                            // 84.78261
	"css-display":                        "[style*=\"display:\"]",                                                // 55.555553
	"css-filter":                         "[style*=\"filter:\"]",                                                 // 50
	"css-flex-direction":                 "[style*=\"flex-direction:\"]",                                         // 50
	"css-flex-wrap":                      "[style*=\"flex-wrap:\"]",                                              // 49.09091
	"css-float":                          "[style*=\"float:\"]",                                                  // 85.10638
	"css-font-kerning":                   "[style*=\"font-kerning:\"]",                                           // 66.666664
	"css-font-weight":                    "[style*=\"font-weight:\"]",                                            // 76.666664
	"css-font":                           "[style*=\"font:\"]",                                                   // 95.833336
	"css-gap":                            "[style*=\"gap:\"]",                                                    // 40
	"css-grid-template":                  "[style*=\"grid-template:\"]",                                          // 34.042553
	"css-height":                         "[style*=\"height:\"], [height]",                                       // 77.08333
	"css-hyphens":                        "[style*=\"hyphens:\"]",                                                // 31.111107
	"css-important":                      "[style*=\"!important\"]",                                              // 43.478264
	"css-inline-size":                    "[style*=\"inline-size:\"]",                                            // 43.478264
	"css-intrinsic-size":                 "[style*=\"intrinsic-size:\"]",                                         // 40.54054
	"css-justify-content":                "[style*=\"justify-content:\"]",                                        // 59.25926
	"css-letter-spacing":                 "[style*=\"letter-spacing:\"]",                                         // 87.23404
	"css-line-height":                    "[style*=\"line-height:\"]",                                            // 82.608696
	"css-list-style-image":               "[style*=\"list-style-image:\"]",                                       // 54.16667
	"css-list-style-position":            "[style*=\"list-style-position:\"]",                                    // 87.5
	"css-list-style":                     "[style*=\"list-style:\"]",                                             // 62.500004
	"css-margin-block-start-end":         "[style*=\"margin-block-start:\"], [style*=\"margin-block-end:\"]",     // 32.07547
	"css-margin-inline-block":            "[style*=\"margin-inline-block:\"]",                                    // 16.981125
	"css-margin-inline-start-end":        "[style*=\"margin-inline-start:\"], [style*=\"margin-inline-end:\"]",   // 32.07547
	"css-margin-inline":                  "[style*=\"margin-inline:\"]",                                          // 43.39623
	"css-margin":                         "[style*=\"margin:\"]",                                                 // 71.42857
	"css-max-block-size":                 "[style*=\"max-block-size:\"]",                                         // 35.714287
	"css-max-height":                     "[style*=\"max-height:\"]",                                             // 86.95652
	"css-max-width":                      "[style*=\"max-width:\"]",                                              // 76.47058
	"css-min-height":                     "[style*=\"min-height:\"]",                                             // 82.608696
	"css-min-inline-size":                "[style*=\"min-inline-size:\"]",                                        // 33.33333
	"css-min-width":                      "[style*=\"min-width:\"]",                                              // 86.95652
	"css-mix-blend-mode":                 "[style*=\"mix-blend-mode:\"]",                                         // 62.745094
	"css-modern-color":                   "[style*=\"modern-color:\"]",                                           // 10.81081
	"css-object-fit":                     "[style*=\"object-fit:\"]",                                             // 57.142857
	"css-object-position":                "[style*=\"object-position:\"]",                                        // 55.10204
	"css-opacity":                        "[style*=\"opacity:\"]",                                                // 63.04348
	"css-outline-offset":                 "[style*=\"outline-offset:\"]",                                         // 42.5
	"css-outline":                        "[style*=\"outline:\"]",                                                // 80.85106
	"css-overflow-wrap":                  "[style*=\"overflow-wrap:\"]",                                          // 6.6666603
	"css-overflow":                       "[style*=\"overflow:\"]",                                               // 78.26087
	"css-padding-block-start-end":        "[style*=\"padding-block-start:\"], [style*=\"padding-block-end:\"]",   // 32.07547
	"css-padding-inline-block":           "[style*=\"padding-inline-block:\"]",                                   // 16.981125
	"css-padding-inline-start-end":       "[style*=\"padding-inline-start:\"], [style*=\"padding-inline-end:\"]", // 32.07547
	"css-padding":                        "[style*=\"padding:\"], [padding]",                                     // 87.755104
	"css-position":                       "[style*=\"position:\"]",                                               // 19.56522
	"css-radial-gradient":                "[style*=\"radial-gradient:\"]",                                        // 64.583336
	"css-rgb":                            "[style*=\"rgb(\"]",                                                    // 53.846153
	"css-rgba":                           "[style*=\"rgba(\"]",                                                   // 56
	"css-scroll-snap":                    "[style*=\"roll-snap:\"]",                                              // 38.88889
	"css-tab-size":                       "[style*=\"tab-size:\"]",                                               // 32.075474
	"css-table-layout":                   "[style*=\"table-layout:\"]",                                           // 53.33333
	"css-text-align-last":                "[style*=\"text-align-last:\"]",                                        // 42.307693
	"css-text-align":                     "[style*=\"text-align:\"]",                                             // 60.416664
	"css-text-decoration-color":          "[style*=\"text-decoration-color:\"]",                                  // 67.34695
	"css-text-decoration-thickness":      "[style*=\"text-decoration-thickness:\"]",                              // 38.333336
	"css-text-decoration":                "[style*=\"text-decoration:\"]",                                        // 67.391304
	"css-text-emphasis-position":         "[style*=\"text-emphasis-position:\"]",                                 // 28.571434
	"css-text-emphasis":                  "[style*=\"text-emphasis:\"]",                                          // 36.734695
	"css-text-indent":                    "[style*=\"text-indent:\"]",                                            // 78.43137
	"css-text-overflow":                  "[style*=\"text-overflow:\"]",                                          // 58.695656
	"css-text-shadow":                    "[style*=\"text-shadow:\"]",                                            // 69.565216
	"css-text-transform":                 "[style*=\"text-transform:\"]",                                         // 86.666664
	"css-text-underline-offset":          "[style*=\"text-underline-offset:\"]",                                  // 39.285713
	"css-transform":                      "[style*=\"transform:\"]",                                              // 50
	"css-unit-calc":                      "[style*=\"calc(:\"]",                                                  // 56.25
	"css-variables":                      "[style*=\"variables:\"]",                                              // 46.551727
	"css-visibility":                     "[style*=\"visibility:\"]",                                             // 52.173916
	"css-white-space":                    "[style*=\"white-space:\"]",                                            // 58.69565
	"css-width":                          "[style*=\"width:\"], [width]",                                         // 87.5
	"css-word-break":                     "[style*=\"word-break:\"]",                                             // 28.888887
	"css-writing-mode":                   "[style*=\"writing-mode:\"]",                                           // 56.25
	"css-z-index":                        "[style*=\"z-index:\"]",                                                // 76.08696
}

// some CSS tests using regex for things that can't be merged inline
var cssRegexpTests = map[string]*regexp.Regexp{
	"css-at-font-face":                  regexp.MustCompile(`(?mi)@font\-face\s+?{`),        // 26.923073
	"css-at-import":                     regexp.MustCompile(`(?mi)@import\s`),               // 36.170216
	"css-at-keyframes":                  regexp.MustCompile(`(?mi)@keyframes\s`),            // 31.914898
	"css-at-media":                      regexp.MustCompile(`(?mi)@media\s?\(`),             // 47.05882
	"css-at-supports":                   regexp.MustCompile(`(?mi)@supports\s?\(`),          // 40.81633
	"css-pseudo-class-active":           regexp.MustCompile(`:active`),                      // 52.173912
	"css-pseudo-class-checked":          regexp.MustCompile(`:checked`),                     // 31.91489
	"css-pseudo-class-first-child":      regexp.MustCompile(`:first\-child`),                // 66.666664
	"css-pseudo-class-first-of-type":    regexp.MustCompile(`:first\-of\-type`),             // 62.5
	"css-pseudo-class-focus":            regexp.MustCompile(`:focus`),                       // 47.826088
	"css-pseudo-class-has":              regexp.MustCompile(`:has`),                         // 25.531914
	"css-pseudo-class-hover":            regexp.MustCompile(`:hover`),                       // 60.41667
	"css-pseudo-class-lang":             regexp.MustCompile(`:lang\s?\(`),                   // 18.918922
	"css-pseudo-class-last-child":       regexp.MustCompile(`:last\-child`),                 // 64.58333
	"css-pseudo-class-last-of-type":     regexp.MustCompile(`:last\-of\-type`),              // 60.416664
	"css-pseudo-class-link":             regexp.MustCompile(`:link`),                        // 81.63265
	"css-pseudo-class-not":              regexp.MustCompile(`:not(\s+)?\(`),                 // 44.89796
	"css-pseudo-class-nth-child":        regexp.MustCompile(`:nth\-child(\s+)?\(`),          // 44.89796
	"css-pseudo-class-nth-last-child":   regexp.MustCompile(`:nth\-last\-child(\s+)?\(`),    // 44.89796
	"css-pseudo-class-nth-last-of-type": regexp.MustCompile(`:nth\-last\-of\-type(\s+)?\(`), // 42.857143
	"css-pseudo-class-nth-of-type":      regexp.MustCompile(`:nth\-of\-type(\s+)?\(`),       // 42.857143
	"css-pseudo-class-only-child":       regexp.MustCompile(`:only\-child(\s+)?\(`),         // 64.58333
	"css-pseudo-class-only-of-type":     regexp.MustCompile(`:only\-of\-type(\s+)?\(`),      // 64.58333
	"css-pseudo-class-target":           regexp.MustCompile(`:target`),                      // 39.13044
	"css-pseudo-class-visited":          regexp.MustCompile(`:visited`),                     // 39.13044
	"css-pseudo-element-after":          regexp.MustCompile(`:after`),                       // 40
	"css-pseudo-element-before":         regexp.MustCompile(`:before`),                      // 40
	"css-pseudo-element-first-letter":   regexp.MustCompile(`::first\-letter`),              // 60
	"css-pseudo-element-first-line":     regexp.MustCompile(`::first\-line`),                // 60
	"css-pseudo-element-marker":         regexp.MustCompile(`::marker`),                     // 50
	"css-pseudo-element-placeholder":    regexp.MustCompile(`::placeholder`),                // 32
}

// some CSS tests using regex for units
var cssRegexpUnitTests = map[string]*regexp.Regexp{
	"css-unit-ch":      regexp.MustCompile(`\b\d+ch\b`),     // 66.666664
	"css-unit-initial": regexp.MustCompile(`:\s?initial\b`), // 58.33333
	"css-unit-rem":     regexp.MustCompile(`\b\d+rem\b`),    // 66.666664
	"css-unit-vh":      regexp.MustCompile(`\b\d+vh\b`),     // 68.75
	"css-unit-vmax":    regexp.MustCompile(`\b\d+vmax\b`),   // 60.416664
	"css-unit-vmin":    regexp.MustCompile(`\b\d+vmin\b`),   // 58.333336
	"css-unit-vw":      regexp.MustCompile(`\b\d+vw\b`),     // 77.08333
}
