document.addEventListener('DOMContentLoaded', e => {

});



let selecting = false;
function select_song(winner, loser) {
	if (selecting) {
		return;
	}
	selecting = true;

	fetch(`/api/select_song?winner=${winner}&loser=${loser}`, { method: 'POST' })
		.then(() => window.location.reload());
	selecting = false;
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
