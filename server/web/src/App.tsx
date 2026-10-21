import { lazy, Suspense } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { SessionProvider, useSession } from './lib/session'
import { ErrorState, PageLoader, ToastProvider } from './components/ui'
import { Layout } from './components/Layout'
import { Home } from './pages/Home'
import { Login } from './pages/Login'
import { Setup } from './pages/Setup'
import { Library } from './pages/Library'
import { MovieDetail } from './pages/MovieDetail'
import { SeriesDetail } from './pages/SeriesDetail'
import { Search } from './pages/Search'
import { Collections, CollectionPage } from './pages/Collections'
import { MyList } from './pages/MyList'
import { Profile } from './pages/Profile'
import { LinkDevice } from './pages/Link'

// Le lecteur (hls.js) et l'administration ne sont chargés qu'à la demande.
const Player = lazy(() => import('./pages/Player'))
const Admin = lazy(() => import('./pages/admin/Admin'))

function Gate() {
  const { status, user, error, refresh } = useSession()
  if (error && !status) return <ErrorState error={{ message: error }} retry={refresh} />
  if (!status) return <PageLoader />
  if (!status.hasAdmin) return <Setup />
  if (!user) return <Login />
  if (user.role === 'admin' && !status.setupDone) return <Setup />
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/play/:id" element={<Player />} />
        <Route element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="movies" element={<Library kind="movie" />} />
          <Route path="movies/:id" element={<MovieDetail />} />
          <Route path="series" element={<Library kind="series" />} />
          <Route path="series/:id" element={<SeriesDetail />} />
          <Route path="collections" element={<Collections />} />
          <Route path="collections/:id" element={<CollectionPage />} />
          <Route path="mylist" element={<MyList />} />
          <Route path="search" element={<Search />} />
          <Route path="profile" element={<Profile />} />
          <Route path="link" element={<LinkDevice />} />
          <Route path="admin/*" element={user.role === 'admin' ? <Admin /> : <Navigate to="/" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </Suspense>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <SessionProvider>
          <Gate />
        </SessionProvider>
      </ToastProvider>
    </BrowserRouter>
  )
}
