// ============================================================
// content/hardware-bridge.js
// Web Serial & Web HID Hardware Bridge for ERP Companion
// Enables direct USB/Serial barcode scanners and scales
// Ref: ADR-012 (Enterprise Chrome Extension Companion)
// ============================================================

(function () {
  'use strict';

  let activeSerialPort = null;
  let serialReader = null;
  let activeHidDevice = null;

  /**
   * Check browser hardware API support
   */
  function checkHardwareSupport() {
    return {
      hasSerial: typeof navigator !== 'undefined' && 'serial' in navigator,
      hasHid: typeof navigator !== 'undefined' && 'hid' in navigator,
    };
  }

  /**
   * Request and connect to a Web Serial barcode scanner or device
   */
  async function connectSerialScanner(baudRate = 9600) {
    if (!checkHardwareSupport().hasSerial) {
      throw new Error('Web Serial API is not supported in this browser environment');
    }

    try {
      activeSerialPort = await navigator.serial.requestPort();
      await activeSerialPort.open({ baudRate });

      const textDecoder = new TextDecoderStream();
      activeSerialPort.readable.pipeTo(textDecoder.writable);
      serialReader = textDecoder.readable.getReader();

      listenToSerialStream();
      return { success: true, status: 'connected', baudRate };
    } catch (err) {
      console.error('[ERP Hardware Bridge] Serial connection error:', err);
      throw err;
    }
  }

  /**
   * Read incoming chunks from Serial stream and emit scanned barcode events
   */
  async function listenToSerialStream() {
    let buffer = '';
    try {
      while (serialReader) {
        const { value, done } = await serialReader.read();
        if (done) {
          break;
        }
        if (value) {
          buffer += value;
          if (buffer.includes('\n') || buffer.includes('\r')) {
            const barcode = buffer.trim();
            buffer = '';
            if (barcode.length > 0) {
              dispatchScannedEvent(barcode, 'SERIAL');
            }
          }
        }
      }
    } catch (err) {
      console.warn('[ERP Hardware Bridge] Serial read loop ended:', err);
    }
  }

  /**
   * Connect to a Web HID barcode device
   */
  async function connectHidDevice() {
    if (!checkHardwareSupport().hasHid) {
      throw new Error('Web HID API is not supported in this browser environment');
    }

    try {
      const devices = await navigator.hid.requestDevice({ filters: [] });
      if (!devices || devices.length === 0) {
        throw new Error('No HID device selected');
      }

      activeHidDevice = devices[0];
      await activeHidDevice.open();

      activeHidDevice.addEventListener('inputreport', (event) => {
        const { data } = event;
        const decoder = new TextDecoder();
        const str = decoder.decode(data).replace(/\0/g, '').trim();
        if (str.length > 0) {
          dispatchScannedEvent(str, 'HID');
        }
      });

      return { success: true, productName: activeHidDevice.productName };
    } catch (err) {
      console.error('[ERP Hardware Bridge] HID connection error:', err);
      throw err;
    }
  }

  /**
   * Dispatches scanned event to current document and forwards to extension runtime
   */
  function dispatchScannedEvent(barcode, source) {
    if (typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('erp:barcode-scanned', {
          detail: { barcode, source, timestamp: Date.now() },
        })
      );
    }

    if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.sendMessage) {
      chrome.runtime.sendMessage({
        action: 'HARDWARE_BARCODE_SCANNED',
        barcode,
        source,
        timestamp: Date.now(),
      }).catch(() => {});
    }
  }

  // --- Message listener from Side Panel ---
  if (typeof chrome !== 'undefined' && chrome.runtime && chrome.runtime.onMessage) {
    chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
      const msgType = msg?.action || msg?.type;

      if (msgType === 'CHECK_HARDWARE_SUPPORT') {
        sendResponse({ success: true, support: checkHardwareSupport() });
        return false;
      }

      if (msgType === 'CONNECT_SERIAL') {
        connectSerialScanner(msg.baudRate || 9600)
          .then((res) => sendResponse(res))
          .catch((err) => sendResponse({ success: false, error: err.message }));
        return true;
      }

      if (msgType === 'CONNECT_HID') {
        connectHidDevice()
          .then((res) => sendResponse(res))
          .catch((err) => sendResponse({ success: false, error: err.message }));
        return true;
      }

      return false;
    });
  }

  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      checkHardwareSupport,
      dispatchScannedEvent,
    };
  }

  if (typeof console !== 'undefined' && console.log) {
    console.log('[ERP Companion] Hardware Bridge (Web Serial / Web HID) initialized.');
  }
})();

