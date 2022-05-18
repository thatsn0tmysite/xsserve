/** CLIPBOARD **/
function copyToClipboard(element) {
	//Must be input element!
	element.select();
	element.setSelectionRange(0, 99999);
	document.execCommand("copy");
}

/** EVILJS MODAL **/
function sendEvilJs() {
	var code = document.getElementById("eviljs").value;
	//var result = document.getElementById("eviljs-result").value;
	var trigger_id = document.getElementById("trigger-id").value;

	var http = new XMLHttpRequest();
	var url = `/send/js?id=${trigger_id}`;
	var params = `code=${code}`
	http.open("POST", url, true);
	http.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	http.onreadystatechange = function () {
		if (http.readyState == 4 && http.status == 200) {
			console.log(http.responseText)
		}
	};
	http.send(params);
}

/** MODAL **/
// Modal Config
const isOpenClass = "modal-is-open";
const openingClass = "modal-is-opening";
const closingClass = "modal-is-closing";
const animationDuration = 400; // ms
let visibleModal = null;

// Toggle modal
const toggleModal = (event) => {
	event.preventDefault();
	const modal = document.getElementById(
		event.target.getAttribute("data-target")
	);
	typeof modal != "undefined" && modal != null && isModalOpen(modal)
		? closeModal(modal)
		: openModal(modal);
};

// Is modal open
const isModalOpen = (modal) => {
	return modal.hasAttribute("open") && modal.getAttribute("open") != "false"
		? true
		: false;
};

// Open modal
const openModal = (modal) => {
	if (isScrollbarVisible()) {
		document.documentElement.style.setProperty(
			"--scrollbar-width",
			`${getScrollbarWidth()}px`
		);
	}
	document.documentElement.classList.add(isOpenClass, openingClass);
	setTimeout(() => {
		visibleModal = modal;
		document.documentElement.classList.remove(openingClass);
	}, animationDuration);
	modal.setAttribute("open", true);
};

// Close modal
const closeModal = (modal) => {
	visibleModal = null;
	document.documentElement.classList.add(closingClass);
	setTimeout(() => {
		document.documentElement.classList.remove(closingClass, isOpenClass);
		document.documentElement.style.removeProperty("--scrollbar-width");
		modal.removeAttribute("open");
	}, animationDuration);
};

// Close with a click outside
document.addEventListener("click", (event) => {
	if (visibleModal != null) {
		const modalContent = visibleModal.querySelector("article");
		const isClickInside = modalContent.contains(event.target);
		!isClickInside && closeModal(visibleModal);
	}
});

// Close with Esc key
document.addEventListener("keydown", (event) => {
	if (event.key === "Escape" && visibleModal != null) {
		closeModal(visibleModal);
	}
});

// Get scrollbar width
const getScrollbarWidth = () => {
	// Creating invisible container
	const outer = document.createElement("div");
	outer.style.visibility = "hidden";
	outer.style.overflow = "scroll"; // forcing scrollbar to appear
	outer.style.msOverflowStyle = "scrollbar"; // needed for WinJS apps
	document.body.appendChild(outer);

	// Creating inner element and placing it in the container
	const inner = document.createElement("div");
	outer.appendChild(inner);

	// Calculating difference between container's full width and the child width
	const scrollbarWidth = outer.offsetWidth - inner.offsetWidth;

	// Removing temporary elements from the DOM
	outer.parentNode.removeChild(outer);

	return scrollbarWidth;
};

// Is scrollbar visible
const isScrollbarVisible = () => {
	return document.body.scrollHeight > screen.height;
};
