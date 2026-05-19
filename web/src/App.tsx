import { useEffect, useState } from "react";
import {
  NavLink,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import Survey from "./pages/Survey";
import Plan from "./pages/Plan";
import Replace from "./pages/Replace";
import Shopping from "./pages/Shopping";
import Pricing from "./pages/Pricing";
import History from "./pages/History";
import Family from "./pages/Family";
import Login from "./pages/Login";
import InstallPrompt from "./components/InstallPrompt";
import { clearAuth, loadAuth, type AuthUser } from "./api/auth";

export default function App() {
  const [auth, setAuth] = useState<AuthUser | null | undefined>(undefined);
  const loc = useLocation();
  const nav = useNavigate();

  useEffect(() => {
    loadAuth().then(setAuth);
  }, []);

  // Поллим auth при каждой смене URL — это обновляет header после Login.
  useEffect(() => {
    loadAuth().then(setAuth);
  }, [loc.pathname]);

  const onLogout = async () => {
    await clearAuth();
    setAuth(null);
    nav("/login");
  };

  if (auth === undefined) {
    return null; // короткая загрузка, без мигания UI
  }

  return (
    <>
      <header>
        <h1>План питания</h1>
        {auth ? (
          <nav>
            <NavLink to="/survey">Анкета</NavLink>
            <NavLink to="/plan">План</NavLink>
            <NavLink to="/history">История</NavLink>
            <NavLink to="/shopping">Покупки</NavLink>
            <NavLink to="/family">Семья</NavLink>
            <NavLink to="/pricing">Цена</NavLink>
            <span style={{ marginLeft: 14, fontSize: 13, opacity: 0.85 }}>
              {auth.email}
            </span>
            <a
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onLogout();
              }}
              style={{ marginLeft: 10 }}
            >
              Выйти
            </a>
          </nav>
        ) : (
          <nav>
            <NavLink to="/login">Войти</NavLink>
          </nav>
        )}
      </header>
      <main>
        <Routes>
          <Route
            path="/"
            element={
              <Navigate to={auth ? "/survey" : "/login"} replace />
            }
          />
          <Route
            path="/login"
            element={auth ? <Navigate to="/survey" replace /> : <Login />}
          />

          <Route
            path="/survey"
            element={auth ? <Survey /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/plan"
            element={auth ? <Plan /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/replace"
            element={auth ? <Replace /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/history"
            element={auth ? <History /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/shopping"
            element={auth ? <Shopping /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/family"
            element={auth ? <Family /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/pricing"
            element={auth ? <Pricing /> : <Navigate to="/login" replace />}
          />
        </Routes>
      </main>
      {auth && <InstallPrompt />}
    </>
  );
}
