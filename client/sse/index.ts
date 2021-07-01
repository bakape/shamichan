// Listen to messages from server, execute fn to process message
export default function (fn: Function) {
    window.addEventListener("focus", () => {
        (document.getElementById("favicon") as HTMLLinkElement).href
            = "/assets/favicons/default.ico";
    });
    
    const source = new EventSource("/api/sse");
    source.onmessage = (e) => {
        if (!document.hasFocus()) {
            (document.getElementById("favicon") as HTMLLinkElement).href
                = "/assets/favicons/reply.ico";
        }
        fn(JSON.parse(e.data));
    }
    source.onerror = () => {
        source.close();
    }
}