document.addEventListener('DOMContentLoaded', function(){
    var pageHeight = 1430;
    document.body.style.transform = 'translate3d(0px,0px,0px)';

    document.addEventListener('keyup', checkKey, false)
    function checkKey(e) {
        e = e || window.event;
        if (e.keyCode == '38') {
            scrollPage(pageHeight);
        }
        else if (e.keyCode == '40') {
            scrollPage(-pageHeight);
        }
    }

    function scrollPage(scrollSize) {
      var yPos = getNewYPos(scrollSize);
      document.body.style.transform = 'translate(0px,' + yPos + ')';
    }

    function getNewYPos(add){
      var oldYPos = document.body.style.transform.split(',')[1];
      console.log(oldYPos)
      oldYPos = parseInt(oldYPos.replace(/px/,''));
      var newYPos = oldYPos + add;
      return Math.min(0, newYPos) + 'px';
    }

}, false);
