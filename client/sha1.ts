// asm.js polyfill for the Web Crypto API SHA-1 function

/*
 * Rusha, a JavaScript implementation of the Secure Hash Algorithm, SHA-1,
 * as defined in FIPS PUB 180-1, tuned for high performance with large inputs.
 * (http://github.com/srijs/rusha)
 *
 * Inspired by Paul Johnstons implementation (http://pajhome.org.uk/crypt/md5).
 *
 * Copyright (c) 2013-2016 Sam Rijs (http://awesam.de), Janis Petersons
 * Released under the terms of the MIT license as follows:
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

// Only need the SHA-1 function for now
(window as any).crypto = {
    subtle: {
        digest(_: string, buf: ArrayBuffer): Promise<ArrayBuffer> {
            return new Promise<ArrayBuffer>(function() {
                return new Rusha().digest(buf)
            })
        }
    }
}

const maxChunkLen = 64 * 1024,
    ceilHeapSize = 0x20000

// SHA1 hasher. Call Rusha.digest() with the ArrayBuffer to hash.
class Rusha {
    core: (k: number, x: number) => void
    heap: ArrayBuffer
    h32: Int32Array
    h8: Int8Array

    constructor() {
        // The size of the heap is the sum of:
        // 1. The padded input message size
        // 2. The extended space the algorithm needs (320 byte)
        // 3. The 160 bit state the algorithm uses
        this.heap = new ArrayBuffer(ceilHeapSize)
        this.h32 = new Int32Array(this.heap)
        this.h8 = new Int8Array(this.heap)
    }

    private padChunk(chunkLen: number, msgLen: number) {
        const padChunkLen = padLen(chunkLen),
            view = new Int32Array(this.heap, 0, padChunkLen >> 2)
        padZeroes(view, chunkLen)
        padData(view, chunkLen, msgLen)
        return padChunkLen
    }

    // The digest and digestFrom* interface returns the hash digest
    // as a hex string.
    public digest(buf: ArrayBuffer): ArrayBuffer {
        return this.rawDigest(buf).buffer
    }

    // Calculate the hash digest as an array of 5 32bit integers.
    private rawDigest(str: ArrayBuffer) {
        const msgLen = str.byteLength
        initState(this.heap)

        // Initialize asm.js function
        this.core = RushaCore({ Int32Array, DataView }, {}, this.heap).hash

        let chunkOffset = 0,
            chunkLen = 64 * 1024
        for (
            chunkOffset = 0;
            msgLen > chunkOffset + chunkLen;
            chunkOffset += chunkLen
        ) {
            this.coreCall(str, chunkOffset, chunkLen, msgLen, false)
        }
        this.coreCall(str, chunkOffset, msgLen - chunkOffset, msgLen, true)
        return getRawDigest(this.heap)
    }

    // Initialize and call the RushaCore,
    // assuming an input buffer of length len * 4.
    private coreCall(
        data: ArrayBuffer,
        chunkOffset: number,
        chunkLen: number,
        msgLen: number,
        finalize: boolean,
    ) {
        let padChunkLen = chunkLen
        if (finalize) {
            padChunkLen = this.padChunk(chunkLen, msgLen)
        }
        this.write(data, chunkOffset, chunkLen)
        this.core(padChunkLen, maxChunkLen)
    }

    // Write data to the heap
    private write(data: ArrayBuffer, chunkOffset: number, chunkLen: number) {
        convertBuffer(data, this.h8, this.h32, chunkOffset, chunkLen, 0)
    }
}

// The low-level RushCore module provides the heart of Rusha,
// a high-speed sha1 implementation working on an Int32Array heap.
// At first glance, the implementation seems complicated, however
// with the SHA1 spec at hand, it is obvious this almost a textbook
// implementation that has a few functions hand-inlined and a few loops
// hand-unrolled.
function RushaCore(stdlib: any, foreign: any, heap: ArrayBuffer) {
    'use asm';
    var H = new stdlib.Int32Array(heap);
    function hash(k: number, x: number) {
        // k in bytes
        k = k | 0;
        x = x | 0;
        var i = 0, j = 0, y0 = 0, z0 = 0, y1 = 0, z1 = 0, y2 = 0, z2 = 0, y3 = 0, z3 = 0, y4 = 0, z4 = 0, t0 = 0, t1 = 0;
        y0 = H[x + 320 >> 2] | 0;
        y1 = H[x + 324 >> 2] | 0;
        y2 = H[x + 328 >> 2] | 0;
        y3 = H[x + 332 >> 2] | 0;
        y4 = H[x + 336 >> 2] | 0;
        for (i = 0; (i | 0) < (k | 0); i = i + 64 | 0) {
            z0 = y0;
            z1 = y1;
            z2 = y2;
            z3 = y3;
            z4 = y4;
            for (j = 0; (j | 0) < 64; j = j + 4 | 0) {
                t1 = H[i + j >> 2] | 0;
                t0 = ((y0 << 5 | y0 >>> 27) + (y1 & y2 | ~y1 & y3) | 0) + ((t1 + y4 | 0) + 1518500249 | 0) | 0;
                y4 = y3;
                y3 = y2;
                y2 = y1 << 30 | y1 >>> 2;
                y1 = y0;
                y0 = t0;
                H[k + j >> 2] = t1;
            }
            for (j = k + 64 | 0; (j | 0) < (k + 80 | 0); j = j + 4 | 0) {
                t1 = (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) << 1 | (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) >>> 31;
                t0 = ((y0 << 5 | y0 >>> 27) + (y1 & y2 | ~y1 & y3) | 0) + ((t1 + y4 | 0) + 1518500249 | 0) | 0;
                y4 = y3;
                y3 = y2;
                y2 = y1 << 30 | y1 >>> 2;
                y1 = y0;
                y0 = t0;
                H[j >> 2] = t1;
            }
            for (j = k + 80 | 0; (j | 0) < (k + 160 | 0); j = j + 4 | 0) {
                t1 = (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) << 1 | (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) >>> 31;
                t0 = ((y0 << 5 | y0 >>> 27) + (y1 ^ y2 ^ y3) | 0) + ((t1 + y4 | 0) + 1859775393 | 0) | 0;
                y4 = y3;
                y3 = y2;
                y2 = y1 << 30 | y1 >>> 2;
                y1 = y0;
                y0 = t0;
                H[j >> 2] = t1;
            }
            for (j = k + 160 | 0; (j | 0) < (k + 240 | 0); j = j + 4 | 0) {
                t1 = (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) << 1 | (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) >>> 31;
                t0 = ((y0 << 5 | y0 >>> 27) + (y1 & y2 | y1 & y3 | y2 & y3) | 0) + ((t1 + y4 | 0) - 1894007588 | 0) | 0;
                y4 = y3;
                y3 = y2;
                y2 = y1 << 30 | y1 >>> 2;
                y1 = y0;
                y0 = t0;
                H[j >> 2] = t1;
            }
            for (j = k + 240 | 0; (j | 0) < (k + 320 | 0); j = j + 4 | 0) {
                t1 = (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) << 1 | (H[j - 12 >> 2] ^ H[j - 32 >> 2] ^ H[j - 56 >> 2] ^ H[j - 64 >> 2]) >>> 31;
                t0 = ((y0 << 5 | y0 >>> 27) + (y1 ^ y2 ^ y3) | 0) + ((t1 + y4 | 0) - 899497514 | 0) | 0;
                y4 = y3;
                y3 = y2;
                y2 = y1 << 30 | y1 >>> 2;
                y1 = y0;
                y0 = t0;
                H[j >> 2] = t1;
            }
            y0 = y0 + z0 | 0;
            y1 = y1 + z1 | 0;
            y2 = y2 + z2 | 0;
            y3 = y3 + z3 | 0;
            y4 = y4 + z4 | 0;
        }
        H[x + 320 >> 2] = y0;
        H[x + 324 >> 2] = y1;
        H[x + 328 >> 2] = y2;
        H[x + 332 >> 2] = y3;
        H[x + 336 >> 2] = y4;
    }
    return { hash: hash };
}

// Calculate the length of buffer that the sha1 routine uses
// including the padding.
function padLen(len: number) {
    for (len += 9; len % 64 > 0; len += 1);
    return len
}

function padZeroes(bin: Int32Array, len: number) {
    for (let i = len >> 2; i < bin.length; i++) {
        bin[i] = 0
    }
}

function padData(bin: Int32Array, chunkLen: number, msgLen: number) {
    bin[chunkLen >> 2] |= 128 << 24 - (chunkLen % 4 << 3)
    // To support msgLen >= 2 GiB, use a float division when computing the
    // high 32-bits of the big-endian message length in bits.
    bin[((chunkLen >> 2) + 2 & ~15) + 14] = msgLen / (1 << 29) | 0
    bin[((chunkLen >> 2) + 2 & ~15) + 15] = msgLen << 3
}

// Convert a buffer or array and write it to the heap.
// The buffer or array is expected to only contain elements < 256.
function convertBuffer(
    data: ArrayBuffer,
    H8: Int8Array,
    H32: Int32Array,
    start: number,
    len: number,
    off: number,
) {
    const buf = new Uint8Array(data)
    let i: number
    const om = off % 4,
        lm = len % 4,
        j = len - lm

    if (j > 0) {
        switch (om) {
            case 0:
                H8[off + 3 | 0] = buf[start]
            case 1:
                H8[off + 2 | 0] = buf[start + 1]
            case 2:
                H8[off + 1 | 0] = buf[start + 2]
            case 3:
                H8[off | 0] = buf[start + 3]
        }
    }
    for (i = 4 - om; i < j; i = i += 4 | 0) {
        H32[off + i >> 2] = buf[start + i] << 24 | buf[start + i + 1] << 16 | buf[start + i + 2] << 8 | buf[start + i + 3]
    }
    switch (lm) {
        case 3:
            H8[off + j + 1 | 0] = buf[start + j + 2]
        case 2:
            H8[off + j + 2 | 0] = buf[start + j + 1]
        case 1:
            H8[off + j + 3 | 0] = buf[start + j]
    }
}

function getRawDigest(heap: ArrayBuffer) {
    const io = new Int32Array(heap, maxChunkLen + 320, 5),
        out = new Int32Array(5),
        arr = new DataView(out.buffer)
    arr.setInt32(0, io[0], false)
    arr.setInt32(4, io[1], false)
    arr.setInt32(8, io[2], false)
    arr.setInt32(12, io[3], false)
    arr.setInt32(16, io[4], false)
    return out
}

function initState(heap: ArrayBuffer) {
    const io = new Int32Array(heap, maxChunkLen + 320, 5)
    io[0] = 1732584193
    io[1] = -271733879
    io[2] = -1732584194
    io[3] = 271733878
    io[4] = -1009589776
}
