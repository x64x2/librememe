const imageUpload = document.getElementById('imageUpload');
const memeCanvas = document.getElementById('memeCanvas');
const topTextInput = document.getElementById('topText');
const centerTextInput = document.getElementById('centerText');
const bottomTextInput = document.getElementById('bottomText');
const downloadMemeBtn = document.getElementById('downloadMeme');
const fontSizeInput = document.getElementById('fontSize');
const fontSizeValue = document.getElementById('fontSizeValue');
const fontColorInput = document.getElementById('fontColor');
const ctx = memeCanvas.getContext('2d');
let uploadedImage;

const wikimediaUrl = "https://commons.wikimedia.org/w/api.php";
    const params = {
        action: "query",
        list: "search",
        srsearch: "cat",
        srlimit: 10,
        format: "json"
    };

const image = [
    'https://upload.wikimedia.org/wikipedia/commons/thumb/1/15/Red_Apple.jpg/1200px-Red_Apple.jpg',
    'https://upload.wikimedia.org/wikipedia/commons/thumb/f/f3/Cherry_%282018%29.jpg/1024px-Cherry_%282018%29.jpg',
    // Add more images here...
];

// Fetch Images
fetch(`${wikimediaUrl}?${new URLSearchParams(params).toString()}`)
   .then(response => response.json())
   .then(data => {
         const pages = data.query.search;
         pages.forEach(page => {
             const imageUrl = `https://commons.wikimedia.org/wiki/Special:FilePath/${page.title}`;
             const img = document.createElement('img');
             img.src = imageUrl;
             img.onclick = () => selectImage(imageUrl);
             imageGallery.appendChild(img);
        });
    })
    .catch(error => console.error("Error:", error));

imageUpload.addEventListener('change', (event) => {
    const reader = new FileReader();
    reader.onload = function () {
        const img = new Image();
        img.onload = function () {
            memeCanvas.width = img.width;
            memeCanvas.height = img.height + 100;
            ctx.drawImage(img, 0, 0);
            uploadedImage = img;
            drawText();
        };
        img.src = reader.result;
    };
    reader.readAsDataURL(event.target.files[0]);
});

topTextInput.addEventListener('input', drawText);
centerTextInput.addEventListener('input', drawText);
bottomTextInput.addEventListener('input', drawText);
fontSizeInput.addEventListener('input', updateFontSize);
fontColorInput.addEventListener('input', drawText);

function selectImage(url) {
    selectedImage = url;
    imageUpload.style.display = 'none';
    memeCanvas.style.display = 'block';
}

function updateFontSize() {
    const fontSize = fontSizeInput.value;
    fontSizeValue.textContent = fontSize;
    drawText();
}

function drawText() {
    if (!uploadedImage) return;
    ctx.clearRect(0, 0, memeCanvas.width, memeCanvas.height);
    ctx.drawImage(uploadedImage, 0, 0);

    const fontSize = fontSizeInput.value;
    ctx.fillStyle = fontColorInput.value;
    ctx.strokeStyle = 'black';
    ctx.lineWidth = 4;
    ctx.font = `${fontSize}px Impact`;
    ctx.textAlign = 'center';

    ctx.fillText(topTextInput.value.toUpperCase(), memeCanvas.width / 2, fontSize);
    ctx.strokeText(topTextInput.value.toUpperCase(), memeCanvas.width / 2, fontSize);

    ctx.fillText(centerTextInput.value.toUpperCase(), memeCanvas.width / 2, memeCanvas.height / 2);
    ctx.strokeText(centerTextInput.value.toUpperCase(), memeCanvas.width / 2, memeCanvas.height / 2);

    ctx.fillText(bottomTextInput.value.toUpperCase(), memeCanvas.width / 2, memeCanvas.height - 20);
    ctx.strokeText(bottomTextInput.value.toUpperCase(), memeCanvas.width / 2, memeCanvas.height - 20);
}

downloadMemeBtn.addEventListener('click', () => {
    const link = document.createElement('a');
    link.href = memeCanvas.toDataURL();
    link.download = 'meme.png';
    link.click();
});