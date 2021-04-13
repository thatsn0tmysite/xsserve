function copyToClipboard(element) {
    //Must be input element!
    element.select()
    element.setSelectionRange(0, 99999)
    document.execCommand("copy");
}