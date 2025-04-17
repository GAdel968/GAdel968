let index = 0;
const slides = document.querySelectorAll('.slide');
const totalSlides = slides.length;

document.getElementById('next').addEventListener('click', () => {
    index = getRandomSlideIndex();
    changeSlide();
});

document.getElementById('prev').addEventListener('click', () => {
    index = getRandomSlideIndex();
    changeSlide();
});

function changeSlide() {
    for (let slide of slides) {
        slide.style.opacity = 0;
    }

    slides[index].style.opacity = 1;
}

function getRandomSlideIndex() {
    let randomIndex;
    do {
        randomIndex = Math.floor(Math.random() * totalSlides);
    } while(randomIndex === index);
    return randomIndex;
}

setInterval(() => {
    index = getRandomSlideIndex();
    changeSlide();
}, 5000);
