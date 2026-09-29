const startWriting = document.getElementById("start-writing");

startWriting.onclick = function () {
    window.location.href = "/write";
};
// Scroll reveal: elements with .reveal fade/slide in as they enter view.
(function () {
    var items = document.querySelectorAll(".chapter-card, .book-header, .reviews-section, .post-card");
    if (!("IntersectionObserver" in window) || !items.length) return;

    items.forEach(function (el) { el.classList.add("reveal"); });

    var io = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
            if (entry.isIntersecting) {
                entry.target.classList.add("reveal-visible");
                io.unobserve(entry.target);
            }
        });
    }, { threshold: 0.12 });

    items.forEach(function (el) { io.observe(el); });
})();
