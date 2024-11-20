const current_round_element = document.getElementById('current_round');
const matche_played_element = document.getElementById('matches_played');

const song1_btn_element = document.getElementById('song1_btn');
const song1_img_element = document.getElementById('song1_img');
const song1_svg_element = document.getElementById('song1_svg');
const song1_title_element = document.getElementById('song1_title');
const song1_artists_element = document.getElementById('song1_artists');

const song2_btn_element = document.getElementById('song2_btn');
const song2_img_element = document.getElementById('song2_img');
const song2_svg_element = document.getElementById('song2_svg');
const song2_title_element = document.getElementById('song2_title');
const song2_artists_element = document.getElementById('song2_artists');

function update_page(resp) {
	console.log('update page!');
	console.log(resp);

	current_round_element.innerText = `Current Round: ${resp.round}`;
	matche_played_element.innerText = `Matches played this Round: ${resp.matches}`

	song1_btn_element.setAttribute('winner', resp.song1_id);
	song1_btn_element.setAttribute('loser', resp.song2_id);
	if (resp.song1_image) {
		song1_img_element.setAttribute('src', resp.song1_image);
		song1_img_element.classList.remove('invisible');
		song1_svg_element.classList.add('invisible');
	} else {
		song1_img_element.classList.add('invisible');
		song1_svg_element.classList.remove('invisible');
	}
	song1_title_element.innerText = resp.song1_title;
	song1_artists_element.innerText = resp.song1_artists;

	song2_btn_element.setAttribute('winner', resp.song2_id);
	song2_btn_element.setAttribute('loser', resp.song1_id);
	if (resp.song2_image) {
		song2_img_element.setAttribute('src', resp.song2_image);
		song2_img_element.classList.remove('invisible');
		song2_svg_element.classList.add('invisible');
	} else {
		song2_img_element.classList.add('invisible');
		song2_svg_element.classList.remove('invisible');
	}
	song2_title_element.innerText = resp.song2_title;
	song2_artists_element.innerText = resp.song2_artists;
}

async function fetch_select_song(winner, loser) {
	console.log('fetch song');
	let url = "/api/select_song";
	if (winner || loser) {
		url += `?winner=${winner}&loser=${loser}`;
	}

	const resp = await fetch(url, { method: 'POST' });

	if (resp.redirected) {
		window.location.href = resp.url;
		return;
	}

	if (!resp.ok) {
		return { status: resp.status, error: resp.body };
	}

	return resp.json();
}

document.addEventListener('DOMContentLoaded', async () => {
	console.log('dom content loaded');
	const resp = await fetch_select_song(null, null);

	if (resp.error) {
		console.error(resp.status, resp.error);
		return;
	}

	update_page(resp);
})

async function select_song(element) {
	console.log('select song');
	const resp = await fetch_select_song(element.getAttribute('winner'), element.getAttribute('loser'))

	if (resp.error) {
		console.error(resp.status, resp.error);
		return;
	}

	update_page(resp);
}


async function select_new_playlist() {
	const resp = await fetch('/api/select_new_playlist', { method: 'GET' }).catch(console.error);

	if (resp.redirected) {
		window.location.href = resp.url;
		return;
	}

	if (!resp.ok) {
		console.error('select_new_playlist error');
		return;
	}

	const body = await resp.json();
	const dialog = document.getElementById('select_new_playlist_dialog');

	dialog.innerHTML = `
				<h1>You have too many active sessions (maximum is 3)</h1>
				<p>Which of the sessions should be replaced by this one?</p>
				`

	for (let i = 0; i < body.sessions.length; i++) {
		const button = document.createElement('button');
		button.innerHTML = `
					<h2>${body.sessions[i].playlist}</h2>
					<p>Started: ${body.sessions[i].started}</p>
					<p>Matches completed: ${body.sessions[i].matches_completed}</p>
				`
		button.addEventListener('click', () => {
			fetch('/api/select_new_playlist?delete=' + body.sessions[i].id)
				.then(() => { window.location.href = '/' }).catch(console.error);
			return
		})
		dialog.appendChild(button);
	}

	const cancel = document.createElement('button');
	cancel.innerText = 'Cancel'
	cancel.addEventListener('click', () => {
		dialog.close();
		dialog.innerHTML = "";
	})
	dialog.appendChild(cancel);

	dialog.showModal();
}
