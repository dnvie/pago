(function (window) {
  const translations = {
    en: {
      paymentAddress: "Payment Address:",
      states: {
        awaiting: {
          heading: "Awaiting Payment...",
          info: "Invoice Expiring at<br />{expiration_date}",
          tip_heading: "Want to tip?",
          tip_other: "Other Amount",
          tip_no_tip: "No Tip",
          tip_other_confirm: "Confirm",
          tip_other_cancel: "Cancel",
          tip_other_enter_amount: "Enter amount",
        },
        mempool: {
          heading: "Confirming...",
          info: "Transaction is in the mempool.",
        },
        confirming: {
          heading: "Confirming...",
          info: "{current_confirmations} of {total_confirmations} confirmations",
        },
        confirmed: {
          heading: "Confirmed!",
          info: "Transaction confirmed at<br />{confirmation_date}",
        },
        underpayment: {
          heading: "Underpayment!",
          info: "Your payment was {missing_amount} XMR short. Please send the remaining amount to the same address. The QR code has been updated accordingly.",
        },
        failed: {
          heading: "Payment Failed!",
          info: "",
        },
      },
    },
    de: {
      paymentAddress: "Zahlungsadresse:",
      states: {
        awaiting: {
          heading: "Warte auf Zahlung...",
          info: "Rechnung läuft ab um<br />{expiration_date}",
          tip_heading: "Möchten Sie Trinkgeld geben?",
          tip_other: "Anderer Betrag",
          tip_no_tip: "Kein Trinkgeld",
          tip_other_confirm: "Bestätigen",
          tip_other_cancel: "Abbrechen",
          tip_other_enter_amount: "Geben Sie den Betrag ein",
        },
        mempool: {
          heading: "Bestätigen...",
          info: "Transaktion ist im Mempool.",
        },
        confirming: {
          heading: "Bestätigen...",
          info: "{current_confirmations} von {total_confirmations} Bestätigungen",
        },
        confirmed: {
          heading: "Bestätigt!",
          info: "Transaktion bestätigt um<br />{confirmation_date}",
        },
        underpayment: {
          heading: "Unterbezahlung!",
          info: "Ihrer Zahlung fehlen {missing_amount} XMR. Bitte senden Sie den fehlenden Betrag an obige Adresse. Der QR-Code wurde entsprechend angepasst.",
        },
        failed: {
          heading: "Zahlung fehlgeschlagen!",
          info: "",
        },
      },
    },
    it: {
      paymentAddress: "Indirizzo di pagamento:",
      states: {
        awaiting: {
          heading: "In attesa di pagamento...",
          info: "Fattura in scadenza a<br />{expiration_date}",
          tip_heading: "Vuoi lasciare una mancia?",
          tip_other: "Altro Importo",
          tip_no_tip: "Nessun Suggerimento",
          tip_other_confirm: "Confermare",
          tip_other_cancel: "Cancellare",
          tip_other_enter_amount: "Inserisci l’importo",
        },
        mempool: {
          heading: "Confermando...",
          info: "La transazione è in mempool.",
        },
        confirming: {
          heading: "Confermando...",
          info: "{current_confirmations} di {total_confirmations} conferme",
        },
        confirmed: {
          heading: "Confermata!",
          info: "Transazione confermata alle<br />{confirmation_date}",
        },
        underpayment: {
          heading: "Pagamento insufficiente!",
          info: "Nel tuo pagamento mancano {missing_amount} XMR. Si prega di inviare l’importo residuo all’indirizzo sopra indicato. Il codice QR è stato aggiorno di conseguenza.",
        },
        failed: {
          heading: "Pagamento non riuscito!",
          info: "",
        },
      },
    },
    no: {
      paymentAddress: "Betalingsadresse:",
      states: {
        awaiting: {
          heading: "Venter på betaling...",
          info: "Faktura utløper<br />{expiration_date}",
          tip_heading: "Vil du legge igjen et tips?",
          tip_other: "Ulik Mengde",
          tip_no_tip: "Ingen Tips",
          tip_other_confirm: "Bekrefte",
          tip_other_cancel: "Kansellere",
          tip_other_enter_amount: "Angi beløp",
        },
        mempool: {
          heading: "Bekrefter...",
          info: "Transaksjonen er i mempool.",
        },
        confirming: {
          heading: "Bekrefter...",
          info: "{current_confirmations} av {total_confirmations} bekreftelser",
        },
        confirmed: {
          heading: "Bekreftet!",
          info: "Transaksjonen ble bekreftet<br />{confirmation_date}",
        },
        underpayment: {
          heading: "Underbetaling!",
          info: "Betalingen din mangler {missing_amount} XMR. Vennligst send det utestående beløpet til adressen ovenfor. QR-koden er justert deretter.",
        },
        failed: {
          heading: "Betaling mislyktes!",
          info: "",
        },
      },
    },
  };

  const bottomTemplates = {
    qr_info: `
        <div class="qr_container"></div>
        <div class="inner_container_bottom_text_container">
            <div class="bottom_heading">{heading}</div>
            <div class="bottom_information">{info}</div>
        </div>
    `,
    tip_selection: `
        <div class="tip_container">
            <div class="tip_heading">{tip_heading}</div>
            <div class="tip_button_container">
                <div class="tip_preset_buttons_container">
                    <div class="tip_preset_button">5<span class="percent_symbol">%</span></div>
                    <div class="tip_preset_button">10<span class="percent_symbol">%</span></div>
                    <div class="tip_preset_button">15<span class="percent_symbol">%</span></div>
                </div>
                <div class="tip_button tip_custom_amount" onclick="changeAwaitingState('custom_tip')">{tip_other}</div>
                <div class="tip_button tip_no_tip" onclick="submitNoTip()">{tip_no_tip}</div>
            </div>
        </div>
    `,
    custom_tip: `
        <div class="custom_tip_container tip_container">
            <div class="tip_heading">{tip_heading}</div>
            <div class="tip_button_container">
                <div class="custom_tip_input">
                    <input type="number" placeholder="{tip_other_enter_amount}" />
                </div>
                <div class="tip_button confirm_button" onclick="submitCustomTip()">{tip_other_confirm}</div>
                <div class="tip_button cancel_button" onclick="changeAwaitingState('tip_selection')">{tip_other_cancel}</div>
            </div>
        </div>
    `,
  };

  function formatString(template, variables) {
    return template.replace(/{(\w+)}/g, (match, key) => {
      return typeof variables[key] !== "undefined" ? variables[key] : match;
    });
  }

  function formatFiat(amount) {
    const fixed = parseFloat(amount).toFixed(2);
    const [whole, decimals] = fixed.split(".");
    const withCommas = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
    return withCommas + "." + decimals;
  }

  function setIfChanged(el, html) {
    if (el && el.innerHTML !== html) el.innerHTML = html;
  }

  function formatDate(isoString) {
    if (!isoString) return "";
    const d = new Date(isoString);
    if (isNaN(d.getTime())) return "";
    return d.toLocaleString([], {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    });
  }

  function formatDateTime(isoString) {
    if (!isoString) return "";
    const d = new Date(isoString);
    if (isNaN(d.getTime()) || d.getFullYear() <= 1970) return "";

    return d.toLocaleString([], {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  class PagoCheckout {
    constructor(config) {
      this.container = document.querySelector(config.container);
      this.invoiceId = config.invoiceId;
      this.onSuccess = config.onSuccess || function () {};
      this.apiBase = config.apiBase || "";

      this.isCleared = false;
      this.successFired = false;
      this.qrGenerated = false;
      this.qrLoaded = false;
      this.enableTipping = config.enableTipping || false;
      this.tipProcessed = false;

      this.appState = {
        language: config.language || "en",
        theme: "awaiting",
        awaitingSubState: "qr_info",
      };

      if (!translations[this.appState.language]) {
        this.appState.language = "en";
      }

      if (!this.container) {
        console.error("Pago: Container not found.");
        return;
      }
      if (!this.invoiceId) {
        this.container.innerHTML = `<div style="color:red; font-family:sans-serif;">Error: No Invoice ID provided</div>`;
        return;
      }

      window.__pagoInstance = this;

      this.injectCSS();
      this.renderSkeleton();

      this.resizeHandler = () => this.applyDynamicScaling();
      window.addEventListener("resize", this.resizeHandler);
      this.applyDynamicScaling();

      this.loadQRLibrary(() => {
        this.qrLoaded = true;
        this.startPolling();
      });
    }

    injectCSS() {
      if (document.getElementById("pago-styles")) return;
      const style = document.createElement("style");
      style.id = "pago-styles";
      style.innerHTML = `@font-face { font-family: 'Inter Variable'; src: url('/assets/fonts/InterVariable.ttf') format('truetype'); font-weight: 100 900; font-style: normal; }
@font-face { font-family: 'Diosevka'; src: url('/assets/fonts/Diosevka-Regular.ttf') format('truetype'); font-weight: 400; font-style: normal; }
@font-face { font-family: 'Diosevka'; src: url('/assets/fonts/Diosevka-Medium.ttf') format('truetype'); font-weight: 500; font-style: normal; }
@font-face { font-family: 'Diosevka Extended'; src: url('/assets/fonts/DiosevkaExtended-Regular.ttf') format('truetype'); font-weight: 400; font-style: normal; }
@font-face { font-family: 'Diosevka Extended'; src: url('/assets/fonts/DiosevkaExtended-Medium.ttf') format('truetype'); font-weight: 500; font-style: normal; }
:root {
    --text: #505050;
    --subtext: #bababa;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #979797 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #e4e4e4 100%);
}

[data-theme="mempool"] {
    --text: #795e31;
    --subtext: #a88e63;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #ffc76e 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #fff5e6 100%);
}

[data-theme="confirming"] {
    --text: #795e31;
    --subtext: #a88e63;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #ffc76e 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #fff5e6 100%);
}

[data-theme="confirmed"] {
    --text: #3e713d;
    --subtext: #a1b599;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #6ac766 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #f2ffe6 100%);
}

[data-theme="underpayment"] {
    --text: #863e33;
    --subtext: #b29595;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #ff673a 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #ffe7e7 100%);
}

[data-theme="failed"] {
    --text: #863e33;
    --subtext: #b29595;
    --border-gradient: linear-gradient(to bottom, #f4f4f4 25%, #ff673a 100%);
    --main-gradient: linear-gradient(to bottom, #ffffff 25%, #ffe7e7 100%);
}

.body {
    width: 100vw;
    height: 100vh;
    margin: 0;
    padding: 0;
    font-feature-settings: "ss03" 1, "cv05" 1, "cv06" 1;
}

.frame {
    width: 100%;
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
    background-color: white;
}

.outer_container {
    width: 230px;
    height: 408px;
    border-radius: 23px;
    background: var(--border-gradient);
    transform: scale(2);
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    align-items: center;
}

.inner_container_header {
    width: 175px;
    height: 27px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 1px;
}

.inner_container_header_element {
    font-family: "Diosevka";
    font-weight: 400;
    font-size: 8px;
    color: #c1c1c1;
    line-height: 2px;
    letter-spacing: 0%;
}

.inner_container {
    width: 224px;
    height: 377px;
    border-radius: 20px;
    background: var(--main-gradient);
    margin-bottom: 3px;
    display: flex;
    flex-direction: column;
    align-items: center;
}

.inner_container_top {
    width: 176px;
    height: 180px;
    border-radius: 20px 20px 0 0;
    border-bottom: 0.5px dashed #ededed;
    display: flex;
    flex-direction: column;
    gap: 9px;
}

.date {
    display: flex;
    align-items: center;
    gap: 3px;
}

.amount_fiat {
    margin-top: 25px;
    display: flex;
    align-items: flex-end;
    gap: 7px;
}

.amount_fiat_number {
    font-family: "Inter Variable";
    font-size: 24px;
    font-weight: 500;
    color: #5d5d5d;
    letter-spacing: -5%;
}

.amount_fiat_currency {
    font-family: "Inter Variable";
    font-size: 10px;
    font-weight: 500;
    color: #dcdcdc;
    letter-spacing: -5%;
    margin-bottom: 4px;
}

.amount_xmr {
    font-family: "Diosevka Extended";
    font-size: 8px;
    font-weight: 500;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    letter-spacing: -2%;
    color: #9d9d9d;
}

.payment_address_heading {
    font-family: "Inter Variable";
    font-weight: 500;
    font-size: 8px;
    letter-spacing: -5%;
    color: #5d5d5d;
}

.payment_address {
    font-family: "Diosevka Extended";
    font-size: 8px;
    letter-spacing: -1%;
    color: #9d9d9d;
    font-weight: 500;
    word-break: break-all;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

.conversion_rate {
    font-family: "Diosevka Extended";
    font-size: 7px;
    letter-spacing: -1%;
    color: #cfcfcf;
    font-weight: 500;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    margin-top: 5px;
}

.inner_container_bottom {
    width: 176px;
    height: 225px;
    border-radius: 0 0 20px 20px;
    display: flex;
    flex-direction: column;
    align-items: center;
}

.qr_container > div, .qr_container svg, .qr_container canvas {
    width: 100% !important;
    height: 100% !important;
    display: flex;
    justify-content: center;
    align-items: center;
}

.qr_container {
    margin-top: 20px;
}

.inner_container_bottom_text_container {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 15px;
}

.bottom_heading {
    font-family: "Inter Variable";
    font-weight: 500;
    font-size: 16px;
    letter-spacing: -5%;
    color: var(--text);
    text-align: center;
}

.bottom_information {
    font-family: "Inter Variable";
    font-size: 8px;
    letter-spacing: -5%;
    color: var(--subtext);
    font-weight: 500;
    text-align: center;
}

.tip_container {
    display: flex;
    flex-direction: column;
    gap: 15px;
    align-items: center;
    margin-top: 20px;
}

.tip_heading {
    font-family: "Inter Variable";
    font-size: 13px;
    letter-spacing: -5%;
    font-weight: 500;
    color: #505050;
}

.tip_button_container {
    display: flex;
    flex-direction: column;
    gap: 7px;
}

.tip_preset_buttons_container {
    width: 176px;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
}

.tip_preset_button {
    width: 55px;
    height: 48px;
    border-radius: 10px;
    background-color: white;
    display: flex;
    gap: 1px;
    justify-content: center;
    align-items: center;
    font-family: "Inter Variable";
    font-size: 14px;
    letter-spacing: -5%;
    font-weight: 500;
    color: #505050;
    box-shadow: 0px 0px 6px -2px rgba(0, 0, 0, 0.01);
}

.tip_preset_button:hover {
    filter: brightness(0.97);
    cursor: pointer;
}

.tip_preset_button:active {
    filter: brightness(0.9);
    transform: translateY(1px);
    cursor: pointer;
}

.percent_symbol {
    font-family: "Diosevka Extended";
    font-size: 14px;
    font-weight: 500;
    color: #cccccc;
    line-height: 1px;
}

.tip_button {
    width: 176px;
    height: 40px;
    border-radius: 10px;
    display: flex;
    justify-content: center;
    align-items: center;
    font-family: "Inter Variable";
    font-size: 11px;
    font-weight: 500;
    letter-spacing: -5%;
    background-color: gray;
    box-shadow: 0px 0px 6px -2px rgba(0, 0, 0, 0.01);
}

.tip_button:hover {
    filter: brightness(0.97);
    cursor: pointer;
}

.tip_button:active {
    filter: brightness(0.9);
    transform: translateY(1px);
    cursor: pointer;
}

.tip_custom_amount {
    background-color: #ffffff;
    color: #5d5d5d;
}

.tip_no_tip {
    background-color: #383838;
    color: #ffffff;
}

.custom_tip_input input {
    width: 176px;
    height: 46px;
    padding: 0;
    margin: 0;
    border: 0.5px solid #f0f0f0;
    border-radius: 10px;
    text-indent: 10px;
    font-family: "Inter Variable";
    font-size: 11px;
    letter-spacing: -5%;
    font-weight: 400;
}

.custom_tip_input input:focus {
    outline: none;
    border: 0.5px solid #d1d1d1;
}

.custom_tip_input input::placeholder {
    color: #d5d5d5;
}

input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
}

input[type="number"] {
    -moz-appearance: textfield;
}

.confirm_button {
    background-color: #84D566;
    color: #0F5C0E;
    border: 0.5px solid #66B450;
}

.cancel_button {
    background-color: #F9604C;
    color: #4C0508;
    border: 0.5px solid #C91C1F;
}

.debug-controls {
    position: fixed;
    bottom: 20px;
    right: 20px;
    display: flex;
    gap: 10px;
    z-index: 9999;
}

.debug-controls button {
    padding: 8px 12px;
    background-color: #333;
    color: #fff;
    border: none;
    border-radius: 6px;
    font-family: sans-serif;
    font-size: 12px;
    cursor: pointer;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.debug-controls button:hover {
    background-color: #555;
}


      .pago-checkout-root .body {
         width: auto;
         height: auto;
         min-height: 450px;
         background: transparent;
      }
      .pago-checkout-root .frame {
         background: transparent;
         transform: none;
      }
      .pago-checkout-root .outer_container {
         margin: 0 auto;
         transform-origin: center center;
         transition: transform 0.1s ease-out;
      }
      .qr_container > div, .qr_container svg, .qr_container canvas {
         width: 100%;
         height: 100%;
         display: flex;
         justify-content: center;
         align-items: center;
      }
      `;
      document.head.appendChild(style);
    }

    renderSkeleton() {
      this.container.innerHTML = `
          <div class="pago-checkout-root">
             <div class="body">
                <div class="frame">
                    <div class="outer_container">
                        <div class="inner_container_header">
                            <div class="inner_container_header_element date">
                                <svg width="5" height="4" viewBox="0 0 5 4" fill="none" xmlns="http://www.w3.org/2000/svg">
                                    <g clip-path="url(#clip0_12_33731)">
                                        <path d="M0 1.2002H5V2.8002H0V1.2002Z" fill="#FAFCF9"/>
                                        <path d="M0 0H5V1.33333H0V0Z" fill="#ED1C24"/>
                                        <path d="M0 2.6665H5V3.99984H0V2.6665Z" fill="#ED1C24"/>
                                    </g>
                                    <defs>
                                        <clipPath id="clip0_12_33731">
                                            <rect width="5" height="4" rx="0.5" fill="white"/>
                                        </clipPath>
                                    </defs>
                                </svg>
                                <span class="inv_date_text">...</span>
                            </div>
                            <div class="inner_container_header_element inv_number"></div>
                        </div>

                        <div class="inner_container" id="pago-inner">
                            <div class="inner_container_top">
                                <div class="amount_fiat">
                                    <div class="amount_fiat_number">...</div>
                                    <div class="amount_fiat_currency">...</div>
                                </div>
                                <div class="amount_xmr">...</div>
                                <div class="payment_address"></div>
                                <div class="conversion_rate"></div>
                            </div>

                            <div class="inner_container_bottom"></div>
                        </div>
                    </div>
                </div>
             </div>
          </div>
        `;
    }

    applyDynamicScaling() {
      if (!this.container) return;
      const outerContainer = this.container.querySelector(".outer_container");
      if (!outerContainer) return;

      const parentWidth = this.container.clientWidth;
      const parentHeight = this.container.clientHeight;

      const targetWidth = parentWidth > 0 ? parentWidth : window.innerWidth;
      const targetHeight = parentHeight > 0 ? parentHeight : window.innerHeight;

      const baseWidth = 230;
      const baseHeight = 408;

      let scaleByHeight = (targetHeight * 0.85) / baseHeight;
      let scaleByWidth = (targetWidth * 0.95) / baseWidth;

      let scale = Math.min(scaleByHeight, scaleByWidth);

      if (scale > 2.5) scale = 2.5;
      else if (scale < 0.8) scale = 0.8;

      outerContainer.style.transform = `scale(${scale})`;
    }

    loadQRLibrary(callback) {
      if (window.QRCodeStyling) {
        callback();
        return;
      }
      const script = document.createElement("script");

      script.src = "/assets/js/qr-code-styling.js";

      script.onload = callback;
      document.head.appendChild(script);
    }

    changeAwaitingState(newState) {
      this.appState.awaitingSubState = newState;
      this.updateUI(this.lastData);
    }

    submitNoTip() {
      this.handleTipSelection(0);
    }

    submitCustomTip() {
      const input = this.container.querySelector(".custom_tip_input input");
      if (input && input.value) {
        this.handleTipSelection(parseFloat(input.value));
      }
    }

    async handleTipSelection(percentage) {
      try {
        const parsedTip = parseInt(percentage);
        if (parsedTip > 0) {
          const response = await fetch(`${this.apiBase}/api/invoice/tip`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              invoice_public_id: this.invoiceId,
              tip_percentage: parsedTip,
            }),
          });

          if (!response.ok) throw new Error("Failed to add tip");
        }

        this.tipProcessed = true;
        this.qrGenerated = false;
        this.appState.awaitingSubState = "qr_info";
        this.fetchStatus();
      } catch (err) {
        console.error(err);
        alert("Tip error. Try again.");
      }
    }

    async fetchStatus() {
      if (this.isCleared) return;
      try {
        const response = await fetch(
          `${this.apiBase}/api/invoice/status?public_id=${this.invoiceId}`,
        );
        if (!response.ok) throw new Error("Invoice not found");
        const data = await response.json();

        this.updateUI(data);

        if (!this.isCleared) {
          const isTippingPhase =
            this.appState.theme === "awaiting" &&
            this.appState.awaitingSubState !== "qr_info";
          if (!isTippingPhase) {
            let nextPollDelay = 2000;
            setTimeout(() => this.fetchStatus(), nextPollDelay);
          } else {
            console.log("Pago: Polling paused while tip screen is active.");
          }
        }
      } catch (err) {
        console.error("Pago API Error:", err);
        if (!this.isCleared) {
          setTimeout(() => this.fetchStatus(), 5000);
        }
      }
    }

    updateUI(data) {
      if (!data) return;
      this.lastData = data;

      let theme = "awaiting";
      if (data.status === "pending") theme = "awaiting";
      else if (data.status === "in_mempool") theme = "mempool";
      else if (data.status === "confirming") theme = "confirming";
      else if (data.status === "confirmed" || data.payment_cleared)
        theme = "confirmed";
      else if (data.status === "underpaid") theme = "underpayment";
      else if (data.status === "expired" || data.status === "failed")
        theme = "failed";

      this.appState.theme = theme;

      const shouldEnableTipping =
        data.tip_enabled !== undefined ? data.tip_enabled : this.enableTipping;

      if (
        shouldEnableTipping &&
        !this.tipProcessed &&
        theme === "awaiting" &&
        data.fiat_amount > 0
      ) {
        if (this.appState.awaitingSubState === "qr_info") {
          this.appState.awaitingSubState = "tip_selection";
        }
      } else if (theme !== "awaiting") {
        this.appState.awaitingSubState = "qr_info";
      }

      const t = translations[this.appState.language];

      const realXmrAmount = data.xmr_amount / 1e12;
      const formattedXmr = realXmrAmount.toFixed(12).replace(/\.?0+$/, "");

      const remainingXmr = (data.xmr_amount - data.amount_received) / 1e12;
      const formattedRemaining = remainingXmr.toFixed(12).replace(/\.?0+$/, "");
      const finalConfirmedAt =
        data.confirmed_at ||
        (theme === "confirmed" ? new Date().toISOString() : "");

      const viewData = {
        invoice_date: formatDate(data.created_at),
        invoice_id: data.invoice_public_id,
        fiat_amount: formatFiat(data.fiat_amount),
        fiat_currency: data.fiat_currency.toUpperCase(),
        xmr_amount: formattedXmr,
        payment_address: data.address,
        conversion_amount: data.exchange_rate,
        expiration_date: formatDateTime(data.expires_at),
        confirmation_date: formatDateTime(finalConfirmedAt),
        current_confirmations: data.current_confs,
        total_confirmations: data.required_confs,
        missing_amount: formattedRemaining,

        ...t,
        ...t.states[this.appState.theme],
      };

      viewData.info = formatString(
        t.states[this.appState.theme].info,
        viewData,
      );
      viewData.heading = t.states[this.appState.theme].heading;

      const innerContainer = this.container.querySelector("#pago-inner");
      if (innerContainer) {
        innerContainer.setAttribute("data-theme", this.appState.theme);
      }

      const outerContainer = this.container.querySelector(".outer_container");
      if (outerContainer) {
        outerContainer.setAttribute("data-theme", this.appState.theme);
      }

      setIfChanged(
        this.container.querySelector(".inv_date_text"),
        " " + viewData.invoice_date,
      );
      setIfChanged(
        this.container.querySelector(".inv_number"),
        "ID: " + viewData.invoice_id,
      );
      setIfChanged(
        this.container.querySelector(".amount_fiat_number"),
        viewData.fiat_amount,
      );
      setIfChanged(
        this.container.querySelector(".amount_fiat_currency"),
        viewData.fiat_currency,
      );
      setIfChanged(
        this.container.querySelector(".amount_xmr"),
        `${viewData.xmr_amount} XMR`,
      );

      const addressHtml = `
            <span class="payment_address_heading">${viewData.paymentAddress || "Payment Address:"}<br /></span>
            ${viewData.payment_address}
        `;
      setIfChanged(
        this.container.querySelector(".payment_address"),
        addressHtml,
      );

      if (viewData.conversion_amount) {
        setIfChanged(
          this.container.querySelector(".conversion_rate"),
          `(1 XMR = ${formatFiat(viewData.conversion_amount)} ${viewData.fiat_currency})`,
        );
      }

      const bottomContainer = this.container.querySelector(
        ".inner_container_bottom",
      );

      if (
        this.appState.theme === "awaiting" ||
        this.appState.theme === "underpayment"
      ) {
        let rawTemplate = bottomTemplates[this.appState.awaitingSubState];
        if (!rawTemplate) rawTemplate = bottomTemplates["qr_info"];

        rawTemplate = rawTemplate.replace(
          /changeAwaitingState/g,
          "window.__pagoInstance.changeAwaitingState",
        );
        rawTemplate = rawTemplate.replace(
          /submitNoTip/g,
          "window.__pagoInstance.submitNoTip",
        );
        rawTemplate = rawTemplate.replace(
          /submitCustomTip/g,
          "window.__pagoInstance.submitCustomTip",
        );

        if (rawTemplate.includes("tip_preset_buttons_container")) {
          rawTemplate = rawTemplate
            .replace(
              '<div class="tip_preset_button">5<span class="percent_symbol">%</span></div>',
              '<div class="tip_preset_button" onclick="window.__pagoInstance.handleTipSelection(5)">5<span class="percent_symbol">%</span></div>',
            )
            .replace(
              '<div class="tip_preset_button">10<span class="percent_symbol">%</span></div>',
              '<div class="tip_preset_button" onclick="window.__pagoInstance.handleTipSelection(10)">10<span class="percent_symbol">%</span></div>',
            )
            .replace(
              '<div class="tip_preset_button">15<span class="percent_symbol">%</span></div>',
              '<div class="tip_preset_button" onclick="window.__pagoInstance.handleTipSelection(15)">15<span class="percent_symbol">%</span></div>',
            );
        }

        bottomContainer.innerHTML = formatString(rawTemplate, viewData);

        if (this.appState.awaitingSubState === "qr_info") {
          const uri = `monero:${data.address}?tx_amount=${this.appState.theme === "underpayment" ? remainingXmr : realXmrAmount}`;
          const qrContainer = this.container.querySelector(".qr_container");
          if (qrContainer) {
            this.renderQR(uri, qrContainer, data.address);
          }
        }
      } else {
        const rawTemplate = bottomTemplates["qr_info"];
        bottomContainer.innerHTML = formatString(rawTemplate, viewData);

        const qrContainer = this.container.querySelector(".qr_container");
        if (qrContainer) {
          const uri = `monero:${data.address}?tx_amount=${realXmrAmount}`;
          this.renderQR(uri, qrContainer, data.address);
        }
      }

      if (theme === "confirmed" || theme === "failed") {
        this.isCleared = true;

        if (theme === "confirmed" && !this.successFired) {
          this.successFired = true;
          this.onSuccess();

          if (data.success_url && data.success_url !== "") {
            setTimeout(() => {
              window.location.href = data.success_url;
            }, 3500);
          }
        }

        if (theme === "failed") {
          setTimeout(() => {
            window.location.reload();
          }, 10000);
        }
      }
    }

    renderQR(uri, container, address) {
      if (this.currentQRUri !== uri || !this.qrCodeElement) {
        this.currentQRUri = uri;
        this.qrCodeElement = document.createElement("div");
        const stylingOptions = {
          type: "svg",
          width: 111,
          height: 111,
          margin: 0,
          data: uri,
          qrOptions: {
            typeNumber: "0",
            mode: "Byte",
            errorCorrectionLevel: "Q",
          },
          imageOptions: {
            saveAsBlob: true,
            hideBackgroundDots: true,
            imageSize: 0.4,
            margin: 0,
          },
          dotsOptions: { type: "rounded", color: "#000000" },
          backgroundOptions: { color: "transparent" },
          cornersSquareOptions: { type: "extra-rounded", color: "#000000" },
          cornersDotOptions: { type: "", color: "#000000" },
          image:
            "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMzIwMSIgaGVpZ2h0PSIzMjAxIiB2aWV3Qm94PSIwIDAgMzIwMSAzMjAxIiBmaWxsPSJub25lIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgo8Y2lyY2xlIGN4PSIxNjAwLjUiIGN5PSIxNjAwLjUiIHI9IjE2MDAuNSIgZmlsbD0id2hpdGUiLz4KPHBhdGggZD0iTTI5NDMgMTYwMEMyOTQzIDIzNDEuMTUgMjM0Mi4yMiAyOTQyIDE2MDEgMjk0MkM4NTkuNzg0IDI5NDIgMjU5IDIzNDEuMTUgMjU5IDE2MDBDMjU5IDg1OC44NiA4NTkuNzk5IDI1OCAxNjAxIDI1OEMyMzQyLjIgMjU4IDI5NDMgODU4LjgwMyAyOTQzIDE2MDBaIiBmaWxsPSJ3aGl0ZSIvPgo8cGF0aCBkPSJNMTYwMC45NyAyNThDODU5Ljk1NiAyNTggMjU4LjA1OCA4NTkuNDM1IDI1OS4wMDEgMTU5OS4zNUMyNTkuMTg3IDE3NDcuMzggMjgyLjgwNiAxODg5Ljc4IDMyNy4xMzYgMjAyMi45OEg3MjguNjgyVjg5NC41NjlMMTYwMC45NyAxNzY2LjM1TDI0NzMuMjEgODk0LjU2OVYyMDIzSDI4NzQuODRDMjkxOS4yNCAxODg5LjgxIDI5NDIuNzMgMTc0Ny40MSAyOTQzIDE1OTkuMzdDMjk0NC4yNiA4NTguNjg1IDIzNDIuMDYgMjU4LjE3OSAxNjAwLjk3IDI1OC4xNzlWMjU4WiIgZmlsbD0iYmxhY2siLz4KPHBhdGggZD0iTTE0MDAuNTQgMTk2OC4yN0wxMDIwLjE5IDE1ODhWMjI5Ny42Nkg3MjkuMzk4TDQ1NSAyMjk3LjcxQzY5MC4zNjkgMjY4My43NiAxMTE1Ljc0IDI5NDIgMTYwMC45NyAyOTQyQzIwODYuMiAyOTQyIDI1MTEuNTkgMjY4My43IDI3NDcgMjI5Ny42NUgyMTgxLjY2VjE1ODhMMTgwMS4yOSAxOTY4LjI3TDE2MDAuOTMgMjE2OC41OEwxNDAwLjU1IDE5NjguMjdIMTQwMC41NFoiIGZpbGw9ImJsYWNrIi8+Cjwvc3ZnPgo=",
        };
        const qrCode = new QRCodeStyling(stylingOptions);
        qrCode.append(this.qrCodeElement);
      }

      container.innerHTML = "";
      container.appendChild(this.qrCodeElement);
      container.style.cursor = "pointer";
      container.onclick = () => {
        navigator.clipboard.writeText(address);
        alert("Address copied!");
      };
    }

    startPolling() {
      this.fetchStatus();
    }

    destroy() {
      this.isCleared = true;
      window.removeEventListener("resize", this.resizeHandler);
      if (this.container) {
        this.container.innerHTML = "";
      }
    }
  }

  window.Pago = {
    mount: function (config) {
      return new PagoCheckout(config);
    },
  };
})(window);
