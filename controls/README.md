# Trackker Control

Interface d'administration web (SolidJS + TypeScript) pour superviser et piloter le backend `trackker` via HTTP.

## Fonctionnalites implementees

- Ecran d'acces par code PIN a 6 chiffres.
- Blocage local pendant 7 jours apres 3 erreurs consecutives (TTL dans `localStorage`).
- Session d'acces en cours memorisee dans `sessionStorage`.
- Dashboard de supervision (SSE temps reel):
  - etat HTTP du backend,
  - nombre de clients connectes,
  - track en cours (titre, artiste, chemin, cover).
- Deck de commandes administrateur avec boutons generiques reutilisables.
- Retour live de la diffusion via `iframe`.

## Configuration

Copie `.env.example` vers `.env` et ajuste selon ton environnement:

```bash
cp .env.example .env
```

Variables supportees:

- `VITE_TRACKKER_API_BASE` (defaut: `http://localhost:9000`)
- `VITE_TRACKKER_PREVIEW_URL` (defaut: meme valeur que `VITE_TRACKKER_API_BASE`)
- `VITE_TRACKKER_CONTROL_PIN` (defaut: `123456`, uniquement fallback local)

## Lancement

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
```

## TODO backend (a brancher cote `core/`)

- `POST /api/control/auth` pour valider le PIN cote serveur.
- `GET /api/control/supervision/events` (SSE) pour `httpOnline`, `connectedClients` et `currentTrack`.
- `POST /api/control/actions/{action}` pour les commandes (`blackout`, `freeze_tracking`, etc.).
