// Client side ordering for live chat messages.
// The server supplies lesson_seq; this module keeps the WebSocket writer fast
// and makes ordering a presentation concern. Missing messages are requested
// through onGap instead of being silently discarded.
(function (root, factory) {
    if (typeof module === 'object' && module.exports) module.exports = factory();
    else root.LiveClassChatOrderBuffer = factory();
}(typeof self !== 'undefined' ? self : this, function () {
    class ChatOrderBuffer {
        constructor({ nextSeq = 0, onMessage, onGap, maxPending = 256 } = {}) {
            this.nextSeq = Number(nextSeq) || 0;
            this.onMessage = onMessage || function () {};
            this.onGap = onGap || function () {};
            this.maxPending = maxPending;
            this.pending = new Map();
            this.seen = new Set();
            this.gapRequested = false;
        }

        push(message) {
            if (!message || !message.message_id) return false;
            if (this.seen.has(message.message_id)) return false;
            const seq = Number(message.lesson_seq) || 0;
            if (seq <= 0) {
                this.seen.add(message.message_id);
                this.onMessage(message);
                return true;
            }
            if (this.nextSeq === 0) this.nextSeq = seq;
            if (seq < this.nextSeq) return false;
            if (this.pending.has(seq)) return false;
            if (seq > this.nextSeq) {
                if (this.pending.size >= this.maxPending) throw new Error('chat ordering buffer full');
                this.pending.set(seq, message);
                if (!this.gapRequested) {
                    this.gapRequested = true;
                    this.onGap(this.nextSeq, seq - 1);
                }
                return true;
            }
            this.emitAndAdvance(message);
            return true;
        }

        // Call this after replaying a gap. It releases all contiguous messages.
        pushReplay(message) { return this.push(message); }

        emitAndAdvance(message) {
            this.seen.add(message.message_id);
            this.onMessage(message);
            this.nextSeq++;
            while (this.pending.has(this.nextSeq)) {
                const next = this.pending.get(this.nextSeq);
                this.pending.delete(this.nextSeq);
                this.seen.add(next.message_id);
                this.onMessage(next);
                this.nextSeq++;
            }
            if (this.pending.size === 0) this.gapRequested = false;
        }
    }
    return ChatOrderBuffer;
}));
