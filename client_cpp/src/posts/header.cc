#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "etc.hh"
#include "models.hh"
#include <array>
#include <string>
#include <unordered_map>

struct code_hash {
    std::size_t operator()(const std::array<char, 2>& c) const
    {
        return (c[0] << 8) | c[1];
    }
};

// Maps country codes to English names
const static std::unordered_map<std::array<char, 2>, std::string, code_hash>
    countries = {
        { { { 'a', 'd' } }, "Andorra" },
        { { { 'a', 'e' } }, "United Arab Emirates" },
        { { { 'a', 'f' } }, "Afghanistan" },
        { { { 'a', 'g' } }, "Antigua and Barbuda" },
        { { { 'a', 'i' } }, "Anguilla" },
        { { { 'a', 'l' } }, "Albania" },
        { { { 'a', 'm' } }, "Armenia" },
        { { { 'a', 'o' } }, "Angola" },
        { { { 'a', 'q' } }, "Antarctica" },
        { { { 'a', 'r' } }, "Argentina" },
        { { { 'a', 's' } }, "American Samoa" },
        { { { 'a', 't' } }, "Austria" },
        { { { 'a', 'u' } }, "Australia" },
        { { { 'a', 'w' } }, "Aruba" },
        { { { 'a', 'x' } }, "Aland Islands !" },
        { { { 'a', 'z' } }, "Azerbaijan" },
        { { { 'b', 'a' } }, "Bosnia and Herzegovina" },
        { { { 'b', 'b' } }, "Barbados" },
        { { { 'b', 'd' } }, "Bangladesh" },
        { { { 'b', 'e' } }, "Belgium" },
        { { { 'b', 'f' } }, "Burkina Faso" },
        { { { 'b', 'g' } }, "Bulgaria" },
        { { { 'b', 'h' } }, "Bahrain" },
        { { { 'b', 'i' } }, "Burundi" },
        { { { 'b', 'j' } }, "Benin" },
        { { { 'b', 'l' } }, "Saint Barthélemy" },
        { { { 'b', 'm' } }, "Bermuda" },
        { { { 'b', 'n' } }, "Brunei Darussalam" },
        { { { 'b', 'o' } }, "Bolivia" },
        { { { 'b', 'q' } }, "Bonaire" },
        { { { 'b', 'r' } }, "Brazil" },
        { { { 'b', 's' } }, "Bahamas" },
        { { { 'b', 't' } }, "Bhutan" },
        { { { 'b', 'v' } }, "Bouvet Island" },
        { { { 'b', 'w' } }, "Botswana" },
        { { { 'b', 'y' } }, "Belarus" },
        { { { 'b', 'z' } }, "Belize" },
        { { { 'c', 'a' } }, "Canada" },
        { { { 'c', 'c' } }, "Cocos (Keeling) Islands" },
        { { { 'c', 'd' } }, "Congo" },
        { { { 'c', 'f' } }, "Central African Republic" },
        { { { 'c', 'g' } }, "Congo" },
        { { { 'c', 'h' } }, "Switzerland" },
        { { { 'c', 'i' } }, "Cote d'Ivoire !" },
        { { { 'c', 'k' } }, "Cook Islands" },
        { { { 'c', 'l' } }, "Chile" },
        { { { 'c', 'm' } }, "Cameroon" },
        { { { 'c', 'n' } }, "China" },
        { { { 'c', 'o' } }, "Colombia" },
        { { { 'c', 'r' } }, "Costa Rica" },
        { { { 'c', 'u' } }, "Cuba" },
        { { { 'c', 'v' } }, "Cabo Verde" },
        { { { 'c', 'w' } }, "Curaçao" },
        { { { 'c', 'x' } }, "Christmas Island" },
        { { { 'c', 'y' } }, "Cyprus" },
        { { { 'c', 'z' } }, "Czechia" },
        { { { 'd', 'e' } }, "Germany" },
        { { { 'd', 'j' } }, "Djibouti" },
        { { { 'd', 'k' } }, "Denmark" },
        { { { 'd', 'm' } }, "Dominica" },
        { { { 'd', 'o' } }, "Dominican Republic" },
        { { { 'd', 'z' } }, "Algeria" },
        { { { 'e', 'c' } }, "Ecuador" },
        { { { 'e', 'e' } }, "Estonia" },
        { { { 'e', 'g' } }, "Egypt" },
        { { { 'e', 'h' } }, "Western Sahara" },
        { { { 'e', 'r' } }, "Eritrea" },
        { { { 'e', 's' } }, "Spain" },
        { { { 'e', 't' } }, "Ethiopia" },
        { { { 'f', 'i' } }, "Finland" },
        { { { 'f', 'j' } }, "Fiji" },
        { { { 'f', 'k' } }, "Falkland Islands (Malvinas)" },
        { { { 'f', 'm' } }, "Micronesia" },
        { { { 'f', 'o' } }, "Faroe Islands" },
        { { { 'f', 'r' } }, "France" },
        { { { 'g', 'a' } }, "Gabon" },
        { { { 'g', 'b' } }, "United Kingdom" },
        { { { 'g', 'd' } }, "Grenada" },
        { { { 'g', 'e' } }, "Georgia" },
        { { { 'g', 'f' } }, "French Guiana" },
        { { { 'g', 'g' } }, "Guernsey" },
        { { { 'g', 'h' } }, "Ghana" },
        { { { 'g', 'i' } }, "Gibraltar" },
        { { { 'g', 'l' } }, "Greenland" },
        { { { 'g', 'm' } }, "Gambia" },
        { { { 'g', 'n' } }, "Guinea" },
        { { { 'g', 'p' } }, "Guadeloupe" },
        { { { 'g', 'q' } }, "Equatorial Guinea" },
        { { { 'g', 'r' } }, "Greece" },
        { { { 'g', 's' } }, "South Georgia and the South Sandwich Islands" },
        { { { 'g', 't' } }, "Guatemala" },
        { { { 'g', 'u' } }, "Guam" },
        { { { 'g', 'w' } }, "Guinea-Bissau" },
        { { { 'g', 'y' } }, "Guyana" },
        { { { 'h', 'k' } }, "Hong Kong" },
        { { { 'h', 'm' } }, "Heard Island and McDonald Islands" },
        { { { 'h', 'n' } }, "Honduras" },
        { { { 'h', 'r' } }, "Croatia" },
        { { { 'h', 't' } }, "Haiti" },
        { { { 'h', 'u' } }, "Hungary" },
        { { { 'i', 'd' } }, "Indonesia" },
        { { { 'i', 'e' } }, "Ireland" },
        { { { 'i', 'l' } }, "Israel" },
        { { { 'i', 'm' } }, "Isle of Man" },
        { { { 'i', 'n' } }, "India" },
        { { { 'i', 'o' } }, "British Indian Ocean Territory" },
        { { { 'i', 'q' } }, "Iraq" },
        { { { 'i', 'r' } }, "Iran" },
        { { { 'i', 's' } }, "Iceland" },
        { { { 'i', 't' } }, "Italy" },
        { { { 'j', 'e' } }, "Jersey" },
        { { { 'j', 'm' } }, "Jamaica" },
        { { { 'j', 'o' } }, "Jordan" },
        { { { 'j', 'p' } }, "Japan" },
        { { { 'k', 'e' } }, "Kenya" },
        { { { 'k', 'g' } }, "Kyrgyzstan" },
        { { { 'k', 'h' } }, "Cambodia" },
        { { { 'k', 'i' } }, "Kiribati" },
        { { { 'k', 'm' } }, "Comoros" },
        { { { 'k', 'n' } }, "Saint Kitts and Nevis" },
        { { { 'k', 'p' } }, "Democratic People's Republic of Korea" },
        { { { 'k', 'r' } }, "Republic of Korea" },
        { { { 'k', 'w' } }, "Kuwait" },
        { { { 'k', 'y' } }, "Cayman Islands" },
        { { { 'k', 'z' } }, "Kazakhstan" },
        { { { 'l', 'a' } }, "Lao People's Democratic Republic" },
        { { { 'l', 'b' } }, "Lebanon" },
        { { { 'l', 'c' } }, "Saint Lucia" },
        { { { 'l', 'i' } }, "Liechtenstein" },
        { { { 'l', 'k' } }, "Sri Lanka" },
        { { { 'l', 'r' } }, "Liberia" },
        { { { 'l', 's' } }, "Lesotho" },
        { { { 'l', 't' } }, "Lithuania" },
        { { { 'l', 'u' } }, "Luxembourg" },
        { { { 'l', 'v' } }, "Latvia" },
        { { { 'l', 'y' } }, "Libya" },
        { { { 'm', 'a' } }, "Morocco" },
        { { { 'm', 'c' } }, "Monaco" },
        { { { 'm', 'd' } }, "Moldova" },
        { { { 'm', 'e' } }, "Montenegro" },
        { { { 'm', 'f' } }, "Saint Martin (French part)" },
        { { { 'm', 'g' } }, "Madagascar" },
        { { { 'm', 'h' } }, "Marshall Islands" },
        { { { 'm', 'k' } }, "Macedonia" },
        { { { 'm', 'l' } }, "Mali" },
        { { { 'm', 'm' } }, "Myanmar" },
        { { { 'm', 'n' } }, "Mongolia" },
        { { { 'm', 'o' } }, "Macao" },
        { { { 'm', 'p' } }, "Northern Mariana Islands" },
        { { { 'm', 'q' } }, "Martinique" },
        { { { 'm', 'r' } }, "Mauritania" },
        { { { 'm', 's' } }, "Montserrat" },
        { { { 'm', 't' } }, "Malta" },
        { { { 'm', 'u' } }, "Mauritius" },
        { { { 'm', 'v' } }, "Maldives" },
        { { { 'm', 'w' } }, "Malawi" },
        { { { 'm', 'x' } }, "Mexico" },
        { { { 'm', 'y' } }, "Malaysia" },
        { { { 'm', 'z' } }, "Mozambique" },
        { { { 'n', 'a' } }, "Namibia" },
        { { { 'n', 'c' } }, "New Caledonia" },
        { { { 'n', 'e' } }, "Niger" },
        { { { 'n', 'f' } }, "Norfolk Island" },
        { { { 'n', 'g' } }, "Nigeria" },
        { { { 'n', 'i' } }, "Nicaragua" },
        { { { 'n', 'l' } }, "Netherlands" },
        { { { 'n', 'o' } }, "Norway" },
        { { { 'n', 'p' } }, "Nepal" },
        { { { 'n', 'r' } }, "Nauru" },
        { { { 'n', 'u' } }, "Niue" },
        { { { 'n', 'z' } }, "New Zealand" },
        { { { 'o', 'm' } }, "Oman" },
        { { { 'p', 'a' } }, "Panama" },
        { { { 'p', 'e' } }, "Peru" },
        { { { 'p', 'f' } }, "French Polynesia" },
        { { { 'p', 'g' } }, "Papua New Guinea" },
        { { { 'p', 'h' } }, "Philippines" },
        { { { 'p', 'k' } }, "Pakistan" },
        { { { 'p', 'l' } }, "Poland" },
        { { { 'p', 'm' } }, "Saint Pierre and Miquelon" },
        { { { 'p', 'n' } }, "Pitcairn" },
        { { { 'p', 'r' } }, "Puerto Rico" },
        { { { 'p', 's' } }, "Palestine" },
        { { { 'p', 't' } }, "Portugal" },
        { { { 'p', 'w' } }, "Palau" },
        { { { 'p', 'y' } }, "Paraguay" },
        { { { 'q', 'a' } }, "Qatar" },
        { { { 'r', 'e' } }, "Reunion !" },
        { { { 'r', 'o' } }, "Romania" },
        { { { 'r', 's' } }, "Serbia" },
        { { { 'r', 'u' } }, "Russian Federation" },
        { { { 'r', 'w' } }, "Rwanda" },
        { { { 's', 'a' } }, "Saudi Arabia" },
        { { { 's', 'b' } }, "Solomon Islands" },
        { { { 's', 'c' } }, "Seychelles" },
        { { { 's', 'd' } }, "Sudan" },
        { { { 's', 'e' } }, "Sweden" },
        { { { 's', 'g' } }, "Singapore" },
        { { { 's', 'h' } }, "Saint Helena" },
        { { { 's', 'i' } }, "Slovenia" },
        { { { 's', 'j' } }, "Svalbard and Jan Mayen" },
        { { { 's', 'k' } }, "Slovakia" },
        { { { 's', 'l' } }, "Sierra Leone" },
        { { { 's', 'm' } }, "San Marino" },
        { { { 's', 'n' } }, "Senegal" },
        { { { 's', 'o' } }, "Somalia" },
        { { { 's', 'r' } }, "Suriname" },
        { { { 's', 's' } }, "South Sudan" },
        { { { 's', 't' } }, "Sao Tome and Principe" },
        { { { 's', 'v' } }, "El Salvador" },
        { { { 's', 'x' } }, "Sint Maarten (Dutch part)" },
        { { { 's', 'y' } }, "Syrian Arab Republic" },
        { { { 's', 'z' } }, "Swaziland" },
        { { { 't', 'c' } }, "Turks and Caicos Islands" },
        { { { 't', 'd' } }, "Chad" },
        { { { 't', 'f' } }, "French Southern Territories" },
        { { { 't', 'g' } }, "Togo" },
        { { { 't', 'h' } }, "Thailand" },
        { { { 't', 'j' } }, "Tajikistan" },
        { { { 't', 'k' } }, "Tokelau" },
        { { { 't', 'l' } }, "Timor-Leste" },
        { { { 't', 'm' } }, "Turkmenistan" },
        { { { 't', 'n' } }, "Tunisia" },
        { { { 't', 'o' } }, "Tonga" },
        { { { 't', 'r' } }, "Turkey" },
        { { { 't', 't' } }, "Trinidad and Tobago" },
        { { { 't', 'v' } }, "Tuvalu" },
        { { { 't', 'w' } }, "Taiwan" },
        { { { 't', 'z' } }, "Tanzania" },
        { { { 'u', 'a' } }, "Ukraine" },
        { { { 'u', 'g' } }, "Uganda" },
        { { { 'u', 'm' } }, "United States Minor Outlying Islands" },
        { { { 'u', 's' } }, "United States of America" },
        { { { 'u', 'y' } }, "Uruguay" },
        { { { 'u', 'z' } }, "Uzbekistan" },
        { { { 'v', 'a' } }, "Holy See" },
        { { { 'v', 'c' } }, "Saint Vincent and the Grenadines" },
        { { { 'v', 'e' } }, "Venezuela" },
        { { { 'v', 'g' } }, "British Virgin Islands" },
        { { { 'v', 'i' } }, "U.S. Virgin Islands" },
        { { { 'v', 'n' } }, "Viet Nam" },
        { { { 'v', 'u' } }, "Vanuatu" },
        { { { 'w', 'f' } }, "Wallis and Futuna" },
        { { { 'w', 's' } }, "Samoa" },
        { { { 'y', 'e' } }, "Yemen" },
        { { { 'y', 't' } }, "Mayotte" },
        { { { 'z', 'a' } }, "South Africa" },
        { { { 'z', 'm' } }, "Zambia" },
        { { { 'z', 'w' } }, "Zimbabwe" },
    };

Node Post::render_header()
{
    Node n = { "header", { { "class", "spaced" } } };
    n.children.reserve(4);

    // TODO: Check if staff, and render moderator checkbox

    if (id == op && !page.thread && page.board == "all") {
        n.children.push_back(
            { "b", { { "class", "board" } }, '/' + board + '/' });
    }
    if (sticky) {
        n.children.push_back({
            "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" },
                { "width", "8" },
                { "height", "8" },
                { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M1.34 0a.5.5 0 0 0 .16 1h.5v2h-1c-.55 0-1 .45-1 1h3v3l.44 1 .56-1v-3h3c0-.55-.45-1-1-1h-1v-2h.5a.5.5 0 1 0 0-1h-4a.5.5 0 0 0-.09 0 .5.5 0 0 0-.06 0z" />)'",
        });
    }
    if (locked) {
        n.children.push_back({
            "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" },
                { "width", "8" },
                { "height", "8" },
                { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M3 0c-1.1 0-2 .9-2 2v1h-1v4h6v-4h-1v-1c0-1.1-.9-2-2-2zm0 1c.56 0 1 .44 1 1v1h-2v-1c0-.56.44-1 1-1z" transform="translate(1)" />)'",
        });
    }

    if (id == op) {
        auto const& subject = threads.at(id).subject;
        std::string s;
        s.reserve(subject.size() + 8); // +2 unicode chars
        s = "「" + subject + "」";
        n.children.push_back({ "h3", s, true });
    }
    n.children.push_back(render_name());
    if (flag && flag->size() == 2) {
        const std::array<char, 2> key = { { (*flag)[0], (*flag)[1] } };
        n.children.push_back({
            "img",
            {
                { "class", "flag" },
                { "src", "/assets/flags/" + *flag + ".svg" },
                {
                    "title",
                    countries.count(key) ? countries.at(key) : *flag,
                },
            },
        });
    }
    n.children.push_back(render_time());

    const auto id_str = std::to_string(id);
    std::string url = "#p" + id_str;
    if (!page.thread) {
        url = absolute_thread_url(id, board) + "?last=100" + url;
    }
    n.children.push_back({
        "nav",
        {},
        {
            {
                "a",
                {
                    { "href", url },
                },
                "No.",
            },
            {
                "a",
                {
                    { "class", "quote" },
                    { "href", url },
                },
                id_str,
            },
        },
    });

    if (id == op && !page.thread && !page.catalog) {
        n.children.push_back({ "span", {},
            brunhild::Children({ render_expand_link(board, id),
                render_last_100_link(board, id) }) });
    }

    n.children.push_back({ "a", { { "class", "control" } },
        R"'(<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8"><path d="M1.5 0l-1.5 1.5 4 4 4-4-1.5-1.5-2.5 2.5-2.5-2.5z" transform="translate(0 1)" /></svg>)'" });

    n.stringify_subtree();
    return n;
}

Node Post::render_name()
{
    Node n("b", { { "class", "name spaced" } });
    if (sage) {
        n.attrs["class"] += " sage";
    }

    if (options.anonymise) {
        n.children = { Node("span", lang.posts.at("anon")) };
        return n;
    }

    if (name || !trip) {
        n.children.push_back(name ? Node("span", *name, true)
                                  : Node("span", lang.posts.at("anon")));
    }
    if (trip) {
        n.children.push_back({ "code", "!" + *trip, true });
    }
    if (poster_id) {
        n.children.push_back({ "span", *poster_id, true });
    }
    if (auth) {
        n.attrs["class"] += " admin";
        n.children.push_back({ "span", "## " + lang.posts.at(*auth) });
    }
    if (post_ids.mine.count(id)) {
        n.children.push_back({ "i", lang.posts.at("you") });
    }

    return n;
}

Node Post::render_time()
{
    using std::setw;

    auto then = std::localtime(&time);

    // Renders classic absolute timestamp
    std::ostringstream abs;
    abs << std::setfill('0') << setw(2) << then->tm_mday << ' '
        << lang.calendar[then->tm_mon] << ' ' << 1900 + then->tm_year << " ("
        << lang.week[then->tm_wday] << ") " << setw(2) << then->tm_hour << ':'
        << setw(2) << then->tm_min;

    const auto rel = relative_time(time);

    return Node("time",
        { { "title", options.relative_time ? abs.str() : rel } },
        options.relative_time ? rel : abs.str());
}
